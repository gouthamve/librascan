package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0002, Down0002)
}

func Up0002(ctx context.Context, tx *sql.Tx) error {
	query := `
ALTER TABLE books
DROP authors;

ALTER TABLE books
DROP categories;

PRAGMA foreign_keys=on;
`

	_, err := tx.ExecContext(ctx, query)

	return err
}

func Down0002(ctx context.Context, tx *sql.Tx) error {
	query := `
	ALTER TABLE books
	ADD COLUMN authors BLOB,
	ADD COLUMN categories BLOB;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}
