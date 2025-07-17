package handlers

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/gouthamve/librascan/migrations"

	"github.com/gouthamve/librascan/pkg/models"
)

func TestDatabase(t *testing.T) {
	// Create a temporary database file
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Create the initial tables
	if err := migrations.Up0001(t.Context(), tx); err != nil {
		t.Fatalf("failed to create initial tables: %v", err)
	}
	if err := migrations.Up0002(t.Context(), tx); err != nil {
		t.Fatalf("failed to create initial tables2: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// Create a new book
	book := models.Book{
		ISBN:          9780141182550,
		Title:         "1984",
		Description:   "A dystopian novel by George Orwell",
		Authors:       []string{"George Orwell"},
		Publisher:     "Penguin Books",
		PublishedDate: "1949",
		Categories:    []string{"Fiction"},
		Language:      "English",
		CoverURL:      "https://covers.openlibrary.org/b/id/7222246-L.jpg",
		Pages:         328,
		ShelfID:       1,
		ShelfName:     "office-big",
		RowNumber:     1,
	}

	// Create Librascan instance with sqlc
	ls := NewLibrascan(db)

	// Insert the book into the database
	if err := ls.storeBook(t.Context(), book); err != nil {
		t.Fatalf("failed to store book: %v", err)
	}

	// Retrieve the book from the database
	book2, err := ls.getBook(t.Context(), int64(book.ISBN))
	if err != nil {
		t.Fatalf("failed to get book: %v", err)
	}

	// Compare the original and retrieved books
	if diff := cmp.Diff(book, book2); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Delete the book from the database
	rows, err := ls.queries.DeleteBook(t.Context(), int64(book.ISBN))
	if err != nil {
		t.Fatalf("failed to delete book: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row to be affected, got %d", rows)
	}

	// Retrieve the book from the database
	book3, err := ls.getBook(t.Context(), int64(book.ISBN))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if cmp.Diff(models.Book{}, book3) != "" {
		t.Fatalf("expected empty book, got %v", book3)
	}
}
