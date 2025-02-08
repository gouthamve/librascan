package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0001, Down0001)
}

func Up0001(ctx context.Context, tx *sql.Tx) error {
	query := `
CREATE TABLE books (
	ISBN INTEGER PRIMARY KEY,
	title TEXT,
	description TEXT,
	authors TEXT,
	categories TEXT,
	publisher TEXT,
	published_date TEXT,
	pages INTEGER,
	language TEXT,
	cover_url TEXT
);

CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
	isbn INTEGER,
	FOREIGN KEY(isbn) REFERENCES books(ISBN)
);

CREATE TABLE categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	isbn INTEGER,
	FOREIGN KEY(isbn) REFERENCES books(ISBN)
);
`

	_, err := tx.ExecContext(ctx, query)
	return err
}

func Down0001(ctx context.Context, tx *sql.Tx) error {
	query := `
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS authors;
DROP TABLE IF EXISTS books;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}
