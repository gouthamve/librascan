package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0003, Down0003)
}

func Up0003(ctx context.Context, tx *sql.Tx) error {
	query := `
    CREATE TABLE people (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		UNIQUE(name)
	);

	CREATE TABLE borrowing (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		isbn INTEGER NOT NULL,
		person_id INTEGER NOT NULL,
		borrowed_at TEXT NOT NULL,
		returned_at TEXT,
		FOREIGN KEY(isbn) REFERENCES books(ISBN),
		FOREIGN KEY(person_id) REFERENCES people(id)
	);
`

	_, err := tx.ExecContext(ctx, query)
	return err
}

func Down0003(ctx context.Context, tx *sql.Tx) error {
	query := `
	DROP TABLE borrowing;
	DROP TABLE people;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}
