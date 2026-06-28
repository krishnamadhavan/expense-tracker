package importers

import "github.com/krishnamadhavan/expense-tracker/internal/ports"

// All returns built-in statement parsers (order = auto-detect priority).
func All() []ports.StatementParser {
	return []ports.StatementParser{
		HDFCBankCSV{},
		ICICIBankCSV{},
		SBIBankCSV{},
		AxisBankCSV{},
		HDFCCreditCardCSV{},
		ICICICreditCardCSV{},
		GenericCSV{},
	}
}

// ByName finds a parser or nil.
func ByName(name string) ports.StatementParser {
	for _, p := range All() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// Detect returns the first matching non-generic parser name.
func Detect(filename string, payload []byte) string {
	for _, p := range All() {
		if p.Name() == "generic_csv" {
			continue
		}
		if p.CanParse(filename, payload) {
			return p.Name()
		}
	}
	return "generic_csv"
}
