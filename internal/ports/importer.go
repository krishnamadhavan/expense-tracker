package ports

import "context"

// ImportBatch is a future bank/UPI import job (stub for M3).
type ImportBatch struct {
	ID     string
	Source string
	Status string
}

// Importer will parse statement files in a later iteration.
type Importer interface {
	// Preview returns row count estimate without committing.
	Preview(ctx context.Context, source string, payload []byte) (rows int, err error)
}

// NoopImporter is a placeholder.
type NoopImporter struct{}

func (NoopImporter) Preview(context.Context, string, []byte) (int, error) { return 0, nil }
