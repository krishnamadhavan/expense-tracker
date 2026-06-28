package importers

import (
	"bytes"
	"encoding/csv"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krishnamadhavan/expense-tracker/internal/domain"
	"github.com/krishnamadhavan/expense-tracker/internal/ports"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func readCSV(payload []byte) ([][]string, error) {
	payload = stripBOM(payload)
	if !utf8.Valid(payload) {
		// try Windows-1252 common in bank exports
		r := transform.NewReader(bytes.NewReader(payload), charmap.Windows1252.NewDecoder())
		b, err := io.ReadAll(r)
		if err == nil {
			payload = b
		}
	}
	cr := csv.NewReader(bytes.NewReader(payload))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true
	var rows [][]string
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

func normHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "/", "")
	return s
}

func findHeaderRow(rows [][]string, needles ...string) (idx int, col map[string]int) {
	col = map[string]int{}
	need := map[string]bool{}
	for _, n := range needles {
		need[normHeader(n)] = true
	}
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		m := map[string]int{}
		hits := 0
		for j, cell := range row {
			h := normHeader(cell)
			if h == "" {
				continue
			}
			m[h] = j
			if need[h] {
				hits++
			}
		}
		// also alias common variants into logical keys
		alias := map[string]string{
			"transactiondate": "date", "txndate": "date", "valuedate": "date", "posteddate": "date",
			"tran date": "date", "trandate": "date", "bookingdate": "date",
			"withdrawalamt": "debit", "withdrawal": "debit", "debitamount": "debit", "dr": "debit",
			"depositamt": "credit", "deposit": "credit", "creditamount": "credit", "cr": "credit",
			"amount": "amount", "transactionamount": "amount",
			"narration": "narration", "description": "narration", "particulars": "narration",
			"remarks": "narration", "transactionremarks": "narration", "merchant": "narration",
			"chqno": "ref", "chequeno": "ref", "refno": "ref", "referenceno": "ref",
			"utr": "ref", "transactionid": "ref", "tranid": "ref", "journalno": "ref",
		}
		logical := map[string]int{}
		for h, j := range m {
			logical[h] = j
			if canon, ok := alias[h]; ok {
				logical[canon] = j
			}
		}
		// score
		score := 0
		if _, ok := logical["date"]; ok {
			score++
		}
		if _, ok := logical["narration"]; ok || hasAny(logical, "narration") {
			score++
		}
		if _, ok := logical["debit"]; ok {
			score++
		}
		if _, ok := logical["credit"]; ok {
			score++
		}
		if _, ok := logical["amount"]; ok {
			score++
		}
		if score >= 2 && (hasAny(logical, "debit", "credit", "amount")) {
			return i, logical
		}
		_ = hits
	}
	return -1, nil
}

func hasAny(m map[string]int, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

func cell(row []string, cols map[string]int, key string) string {
	i, ok := cols[key]
	if !ok || i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func parseAmount(s string) (domain.Money, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "₹", "")
	s = strings.ReplaceAll(s, "Rs.", "")
	s = strings.ReplaceAll(s, "INR", "")
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "0" || s == "0.00" {
		return domain.ZeroMoney, nil
	}
	// (1,234.56) accounting negative
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.Trim(s, "()")
	}
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimPrefix(s, "-")
	}
	m, err := domain.ParseMoney(s)
	if err != nil {
		return domain.Money{}, err
	}
	if neg {
		// direction handled by debit/credit columns; return absolute
	}
	return m, nil
}

var dateLayouts = []string{
	"02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006",
	"2006-01-02", "02 Jan 2006", "2 Jan 2006", "02-Jan-2006", "2-Jan-06",
	"01/02/2006", // US fallback rare
	"2006/01/02", "02.01.2006",
	time.RFC3339,
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	var err error
	for _, layout := range dateLayouts {
		var t time.Time
		t, err = time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, err
}

func rowFromDebitCredit(date, narration, debit, credit, ref string) (ports.ParsedRow, bool, string) {
	d, err := parseDate(date)
	if err != nil || date == "" {
		return ports.ParsedRow{}, false, "bad date"
	}
	deb, _ := parseAmount(debit)
	cred, _ := parseAmount(credit)
	var amt domain.Money
	var dir domain.Direction
	switch {
	case !deb.IsZero() && cred.IsZero():
		amt, dir = deb, domain.DirectionExpense
	case !cred.IsZero() && deb.IsZero():
		amt, dir = cred, domain.DirectionIncome
	case !deb.IsZero() && !cred.IsZero():
		return ports.ParsedRow{}, false, "both debit and credit"
	default:
		return ports.ParsedRow{}, false, "zero amount"
	}
	if amt.IsZero() {
		return ports.ParsedRow{}, false, "zero amount"
	}
	ext := ref
	if ext == "" {
		ext = d.Format("2006-01-02") + "|" + string(dir) + "|" + amt.String() + "|" + narration
	}
	return ports.ParsedRow{
		TxnDate: d, Amount: amt, Direction: dir, PayeeRaw: narration, Memo: narration, ExternalRef: ext,
	}, true, ""
}

func parseTabular(format string, payload []byte) (ports.ParseResult, error) {
	rows, err := readCSV(payload)
	if err != nil && len(rows) == 0 {
		return ports.ParseResult{Format: format}, err
	}
	hi, cols := findHeaderRow(rows)
	res := ports.ParseResult{Format: format, Source: format}
	if hi < 0 {
		res.Warnings = append(res.Warnings, "header row not detected")
		return res, nil
	}
	for _, row := range rows[hi+1:] {
		if isEmptyRow(row) {
			continue
		}
		date := cell(row, cols, "date")
		narr := cell(row, cols, "narration")
		if narr == "" {
			narr = cell(row, cols, "description")
		}
		ref := cell(row, cols, "ref")
		debit := cell(row, cols, "debit")
		credit := cell(row, cols, "credit")
		if debit == "" && credit == "" {
			// single amount column + optional type
			amtS := cell(row, cols, "amount")
			// if amount negative => expense
			amtS = strings.TrimSpace(amtS)
			if amtS == "" {
				res.Skipped++
				continue
			}
			neg := strings.HasPrefix(amtS, "-") || (strings.HasPrefix(amtS, "(") && strings.HasSuffix(amtS, ")"))
			m, err := parseAmount(amtS)
			if err != nil || m.IsZero() {
				res.Skipped++
				continue
			}
			dir := domain.DirectionExpense
			if !neg {
				// ambiguous: treat positive as expense for CC-style "amount" only files unless credit-like narration
				dir = domain.DirectionExpense
			} else {
				dir = domain.DirectionExpense
				// amount already absolute from parseAmount
			}
			// Prefer: if only amount and header says credit for positive — bank files usually have debit/credit
			// For generic amount-only: expense if we can't tell — user can moderate
			d, err := parseDate(date)
			if err != nil {
				res.Skipped++
				continue
			}
			ext := ref
			if ext == "" {
				ext = d.Format("2006-01-02") + "|" + string(dir) + "|" + m.String() + "|" + narr
			}
			res.Rows = append(res.Rows, ports.ParsedRow{
				TxnDate: d, Amount: m, Direction: dir, PayeeRaw: narr, Memo: narr, ExternalRef: ext,
			})
			continue
		}
		pr, ok, why := rowFromDebitCredit(date, narr, debit, credit, ref)
		if !ok {
			res.Skipped++
			if why != "" && len(res.Warnings) < 20 {
				res.Warnings = append(res.Warnings, why+": "+narr)
			}
			continue
		}
		res.Rows = append(res.Rows, pr)
	}
	return res, nil
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func headerContains(payload []byte, snippets ...string) bool {
	// check first 4KB lowercased
	n := 4096
	if len(payload) < n {
		n = len(payload)
	}
	head := strings.ToLower(string(payload[:n]))
	for _, s := range snippets {
		if strings.Contains(head, strings.ToLower(s)) {
			return true
		}
	}
	return false
}
