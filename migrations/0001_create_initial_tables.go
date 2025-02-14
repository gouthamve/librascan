package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0001, Down0001)
}

var (
	Shelfs = []struct {
		Name string
		Rows int
	}{
		{
			Name: "office-big",
			Rows: 6,
		},
		{
			Name: "office-small",
			Rows: 7,
		},
		{
			Name: "home-living-room",
			Rows: 4,
		},
		{
			Name: "home-bedroom-left",
			Rows: 6,
		},
		{
			Name: "home-bedroom-right",
			Rows: 6,
		},
	}
)

func Up0001(ctx context.Context, tx *sql.Tx) error {
	query := `
CREATE TABLE shelfs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	rows_count INTEGER,
	UNIQUE(name)
);

CREATE TABLE books (
	ISBN INTEGER PRIMARY KEY,
	title TEXT,
	description TEXT,
	authors BLOB,
	categories BLOB,
	publisher TEXT,
	published_date TEXT,
	pages INTEGER,
	language TEXT,
	cover_url TEXT,
	shelf_id INTEGER,
	row_number INTEGER,
	FOREIGN KEY(shelf_id) REFERENCES shelfs(id)
);

CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
	isbn INTEGER,
	UNIQUE(name, isbn),
	FOREIGN KEY(isbn) REFERENCES books(ISBN)
);

CREATE TABLE categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	isbn INTEGER,
	UNIQUE(name, isbn),
	FOREIGN KEY(isbn) REFERENCES books(ISBN)
);
`

	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = `INSERT INTO shelfs (id, name, rows_count) VALUES (0, "unknown", 0);`
	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	for _, shelf := range Shelfs {
		query = `INSERT INTO shelfs (name, rows_count) VALUES (?, ?);`
		_, err = tx.ExecContext(ctx, query, shelf.Name, shelf.Rows)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down0001(ctx context.Context, tx *sql.Tx) error {
	query := `
DROP TABLE IF EXISTS shelfs;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS authors;
DROP TABLE IF EXISTS books;
`

	_, err := tx.ExecContext(ctx, query)
	return err
}
