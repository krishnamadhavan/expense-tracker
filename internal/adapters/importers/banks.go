package importers

import (
	"strings"

	"github.com/krishnamadhavan/expense-tracker/internal/ports"
)

type namedCSV struct {
	id       string
	snippets []string
}

func (n namedCSV) Name() string { return n.id }
func (n namedCSV) CanParse(_ string, payload []byte) bool {
	return headerContains(payload, n.snippets...)
}
func (n namedCSV) Parse(_ string, payload []byte) (ports.ParseResult, error) {
	res, err := parseTabular(n.id, payload)
	res.Source = n.id
	res.Format = n.id
	return res, err
}

// HDFCBankCSV — typical "Account Statement" export.
type HDFCBankCSV struct{ namedCSV }

func NewHDFCBank() HDFCBankCSV {
	return HDFCBankCSV{namedCSV{id: "hdfc_bank_csv", snippets: []string{"HDFC", "Withdrawal Amt", "Deposit Amt", "Narration"}}}
}

func (HDFCBankCSV) Name() string { return "hdfc_bank_csv" }
func (h HDFCBankCSV) CanParse(f string, p []byte) bool {
	return headerContains(p, "Withdrawal Amt", "Deposit Amt") || headerContains(p, "withdrawal amt.") ||
		(strings.Contains(strings.ToLower(f), "hdfc") && headerContains(p, "narration"))
}
func (HDFCBankCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	return namedCSV{id: "hdfc_bank_csv"}.Parse(f, p)
}

// ICICIBankCSV
type ICICIBankCSV struct{}

func (ICICIBankCSV) Name() string { return "icici_bank_csv" }
func (ICICIBankCSV) CanParse(f string, p []byte) bool {
	return headerContains(p, "ICICI") || (strings.Contains(strings.ToLower(f), "icici") && !strings.Contains(strings.ToLower(f), "credit"))
}
func (ICICIBankCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	return namedCSV{id: "icici_bank_csv"}.Parse(f, p)
}

// SBIBankCSV
type SBIBankCSV struct{}

func (SBIBankCSV) Name() string { return "sbi_bank_csv" }
func (SBIBankCSV) CanParse(f string, p []byte) bool {
	return headerContains(p, "State Bank") || headerContains(p, "SBI ") || strings.Contains(strings.ToLower(f), "sbi")
}
func (SBIBankCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	return namedCSV{id: "sbi_bank_csv"}.Parse(f, p)
}

// AxisBankCSV
type AxisBankCSV struct{}

func (AxisBankCSV) Name() string { return "axis_bank_csv" }
func (AxisBankCSV) CanParse(f string, p []byte) bool {
	return headerContains(p, "Axis Bank") || strings.Contains(strings.ToLower(f), "axis")
}
func (AxisBankCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	return namedCSV{id: "axis_bank_csv"}.Parse(f, p)
}

// HDFCCreditCardCSV — amount often as purchase (expense) / payment (income/transfer)
type HDFCCreditCardCSV struct{}

func (HDFCCreditCardCSV) Name() string { return "hdfc_cc_csv" }
func (HDFCCreditCardCSV) CanParse(f string, p []byte) bool {
	fl := strings.ToLower(f)
	return headerContains(p, "Credit Card") || strings.Contains(fl, "credit") && strings.Contains(fl, "hdfc") ||
		headerContains(p, "Transaction Description") && headerContains(p, "Reward Point")
}
func (HDFCCreditCardCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	res, err := parseTabular("hdfc_cc_csv", p)
	// CC: credits are often payments — keep as income for P&L or transfer; leave income; user can reclass
	return res, err
}

// ICICICreditCardCSV
type ICICICreditCardCSV struct{}

func (ICICICreditCardCSV) Name() string { return "icici_cc_csv" }
func (ICICICreditCardCSV) CanParse(f string, p []byte) bool {
	fl := strings.ToLower(f)
	return strings.Contains(fl, "icici") && (strings.Contains(fl, "credit") || strings.Contains(fl, "cc")) ||
		headerContains(p, "ICICI Bank Credit Card")
}
func (ICICICreditCardCSV) Parse(f string, p []byte) (ports.ParseResult, error) {
	return parseTabular("icici_cc_csv", p)
}
