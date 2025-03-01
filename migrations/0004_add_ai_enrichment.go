package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0004, Down0004)
}

func Up0004(ctx context.Context, tx *sql.Tx) error {
	query := `
	ALTER TABLE books
	ADD COLUMN is_ai_enriched INTEGER DEFAULT 0;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}

func Down0004(ctx context.Context, tx *sql.Tx) error {
	query := `
	ALTER TABLE books
	DROP COLUMN is_ai_enriched;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}
