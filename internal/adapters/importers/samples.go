package importers

// FormatInfo describes a statement format for UI and sample downloads.
type FormatInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Required    []string `json:"required_columns"`
	Optional    []string `json:"optional_columns"`
	Notes       []string `json:"notes,omitempty"`
	SampleCSV   string   `json:"-"` // body for download
}

// FormatCatalog is user-facing documentation + embedded samples.
func FormatCatalog() []FormatInfo {
	return []FormatInfo{
		{
			ID:          "generic_csv",
			Title:       "Generic bank / card CSV",
			Description: "Works for most exports if you have a date, description, and either separate debit/credit columns or a single amount.",
			Required:    []string{"Date (or Transaction Date / Value Date)", "Narration (or Description / Particulars / Remarks)"},
			Optional:    []string{"Withdrawal/Debit amount", "Deposit/Credit amount", "Amount (single column)", "Ref No / UTR / Cheque No"},
			Notes: []string{
				"Prefer Debit + Credit columns: debit → expense, credit → income.",
				"If only Amount exists, rows are treated as expenses (moderate later if needed).",
				"Header row can be anywhere in the first lines; extra title rows are OK.",
				"Dates: DD/MM/YYYY, YYYY-MM-DD, DD-Mon-YYYY, etc.",
			},
			SampleCSV: sampleGeneric,
		},
		{
			ID:          "hdfc_bank_csv",
			Title:       "HDFC Bank account",
			Description: "Typical HDFC netbanking account statement CSV.",
			Required:    []string{"Date", "Narration", "Withdrawal Amt.", "Deposit Amt."},
			Optional:    []string{"Ref No", "Closing Balance"},
			Notes:       []string{"Auto-detected when headers include Withdrawal Amt / Deposit Amt."},
			SampleCSV:   sampleHDFCBank,
		},
		{
			ID:          "icici_bank_csv",
			Title:       "ICICI Bank account",
			Description: "ICICI account statement-style CSV (column names vary; generic detection applies).",
			Required:    []string{"Date / Transaction Date", "Description / Narration", "Debit / Withdrawal", "Credit / Deposit"},
			Optional:    []string{"Cheque No", "Balance"},
			SampleCSV:   sampleICICIBank,
		},
		{
			ID:          "sbi_bank_csv",
			Title:       "SBI account",
			Description: "State Bank of India statement CSV export.",
			Required:    []string{"Txn Date / Date", "Description", "Debit", "Credit"},
			Optional:    []string{"Ref No", "Balance"},
			SampleCSV:   sampleSBIBank,
		},
		{
			ID:          "axis_bank_csv",
			Title:       "Axis Bank account",
			Description: "Axis Bank statement CSV export.",
			Required:    []string{"Tran Date / Date", "Particulars", "Debit", "Credit"},
			Optional:    []string{"Chq No", "Balance"},
			SampleCSV:   sampleAxisBank,
		},
		{
			ID:          "hdfc_cc_csv",
			Title:       "HDFC Credit Card",
			Description: "Credit card transaction list (purchases as expenses, credits as income).",
			Required:    []string{"Date", "Description / Narration", "Amount or Debit/Credit"},
			Optional:    []string{"Reward points", "Ref"},
			Notes:       []string{"Card payments often appear as credit/income; you can reclassify later."},
			SampleCSV:   sampleHDFCCC,
		},
		{
			ID:          "icici_cc_csv",
			Title:       "ICICI Credit Card",
			Description: "ICICI credit card transactions CSV.",
			Required:    []string{"Date", "Description", "Amount or Debit/Credit"},
			Optional:    []string{"Ref"},
			SampleCSV:   sampleICICICC,
		},
	}
}

// SampleCSV returns sample file body for format id, or empty.
func SampleCSV(formatID string) (filename, body string, ok bool) {
	for _, f := range FormatCatalog() {
		if f.ID == formatID && f.SampleCSV != "" {
			return "sample_" + f.ID + ".csv", f.SampleCSV, true
		}
	}
	// default generic
	if formatID == "" || formatID == "auto" {
		return "sample_generic_csv.csv", sampleGeneric, true
	}
	return "", "", false
}

const sampleGeneric = `Date,Narration,Withdrawal Amt,Deposit Amt,Ref No
15/06/2025,SWIGGY BANGALORE ORDER,450.50,,UTR111
16/06/2025,SALARY NEFT ACME CORP,,75000.00,NEFT999
17/06/2025,UPI-RENT PAYMENT,25000.00,,UPI222
18/06/2025,INTEREST CREDIT,,12.50,INT1
`

const sampleHDFCBank = `Date,Narration,Chq./Ref.No.,Value Dt,Withdrawal Amt.,Deposit Amt.,Closing Balance
01/06/2025,UPI-MERCHANT-SHOP,0000111222,01/06/2025,899.00,,12345.00
02/06/2025,NEFT CR-ACME-SALARY,NEFT998877,02/06/2025,,85000.00,97345.00
`

const sampleICICIBank = `Transaction Date,Description,Withdrawals,Deposits,Balance
03/06/2025,POS 1234 AMAZON,2499.00,,50000.00
04/06/2025,IMPS-INWARD-FRIEND,,2000.00,52000.00
`

const sampleSBIBank = `Txn Date,Description,Debit,Credit,Balance
05/06/2025,ATM WDL,2000.00,,30000.00
06/06/2025,BY TRANSFER, ,1500.00,31500.00
`

const sampleAxisBank = `Tran Date,Particulars,Debit,Credit,Balance
07/06/2025,UPI/phonepe/merchant,350.00,,10000.00
08/06/2025,Salary Credit,,50000.00,60000.00
`

const sampleHDFCCC = `Date,Transaction Description,Amount,Debit/Credit
10/06/2025,FLIPKART INTERNET,1299.00,Debit
11/06/2025,PAYMENT RECEIVED THANK YOU,5000.00,Credit
`

const sampleICICICC = `Date,Description,Amount
12/06/2025,NETFLIX.COM,649.00
13/06/2025,PAYMENT - THANK YOU,-5000.00
`
