package importers

import (
	"testing"

	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

func TestGenericCSV_DebitCredit(t *testing.T) {
	payload := []byte("Date,Narration,Withdrawal Amt,Deposit Amt,Ref No\n" +
		"15/06/2025,SWIGGY BANGALORE,450.50,,UTR123\n" +
		"16/06/2025,SALARY CREDIT,,75000.00,NEFT9\n")
	res, err := GenericCSV{}.Parse("x.csv", payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("rows=%d warn=%v", len(res.Rows), res.Warnings)
	}
	if res.Rows[0].Direction != domain.DirectionExpense || res.Rows[0].Amount.Minor != 45050 {
		t.Fatalf("%+v", res.Rows[0])
	}
	if res.Rows[1].Direction != domain.DirectionIncome || res.Rows[1].Amount.Minor != 7500000 {
		t.Fatalf("%+v", res.Rows[1])
	}
}

func TestDetectHDFC(t *testing.T) {
	payload := []byte("Date,Narration,Withdrawal Amt.,Deposit Amt.\n01/01/2025,x,1,,\n")
	if Detect("stmt.csv", payload) != "hdfc_bank_csv" {
		t.Fatal(Detect("stmt.csv", payload))
	}
}
