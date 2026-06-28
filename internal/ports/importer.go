package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/krishnamadhavan/expense-tracker/internal/domain"
)

// ParsedRow is one statement line after format-specific parsing (pre-DB).
type ParsedRow struct {
	TxnDate     time.Time
	Amount      domain.Money // always >= 0
	Direction   domain.Direction
	PayeeRaw    string
	Memo        string
	ExternalRef string // bank ref / cheque / UTR for dedup
	RawLine     string // debug
}

// ParseResult is the outcome of parsing a file without committing.
type ParseResult struct {
	Source      string
	Format      string
	Rows        []ParsedRow
	Skipped     int
	Warnings    []string
}

// StatementParser turns file bytes into normalized rows for a known format.
type StatementParser interface {
	// Name is the format id, e.g. generic_csv, hdfc_bank_csv, icici_cc_csv.
	Name() string
	// CanParse is a cheap sniff (headers / magic).
	CanParse(filename string, payload []byte) bool
	Parse(filename string, payload []byte) (ParseResult, error)
}

// ImportCommitRequest applies parsed rows into the household ledger.
type ImportCommitRequest struct {
	HouseholdID domain.HouseholdID
	AccountID   domain.AccountID
	// DefaultIncomeStreamID required for income rows when direction=income.
	DefaultIncomeStreamID *domain.IncomeStreamID
	Source                string // e.g. hdfc_bank_csv
	Filename              string
	Rows                  []ParsedRow
	// SkipDuplicates uses external_ref unique constraint (account_id, external_ref).
	SkipDuplicates bool
}

// ImportCommitResult summarizes DB writes.
type ImportCommitResult struct {
	BatchID     uuid.UUID `json:"batch_id"`
	Created     int       `json:"created"`
	SkippedDup  int       `json:"skipped_duplicates"`
	Failed      int       `json:"failed"`
	Errors      []string  `json:"errors,omitempty"`
}

// ImportService is implemented by app/imports (parse + commit).
type ImportService interface {
	ListFormats() []string
	Preview(ctx context.Context, format, filename string, payload []byte) (ParseResult, error)
	// AutoDetect picks the first parser that CanParse, else generic_csv.
	AutoDetect(filename string, payload []byte) string
	Commit(ctx context.Context, req ImportCommitRequest) (ImportCommitResult, error)
}
