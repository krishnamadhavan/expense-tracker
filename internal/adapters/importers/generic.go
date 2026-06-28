package importers

import "github.com/krishnamadhavan/expense-tracker/internal/ports"

// GenericCSV auto-detects columns (date, narration, debit/credit or amount).
type GenericCSV struct{}

func (GenericCSV) Name() string { return "generic_csv" }

func (GenericCSV) CanParse(filename string, payload []byte) bool {
	return true
}

func (GenericCSV) Parse(filename string, payload []byte) (ports.ParseResult, error) {
	res, err := parseTabular("generic_csv", payload)
	res.Source = "generic_csv"
	return res, err
}
