package db

import (
	"database/sql"
	"github.com/gouthamve/librascan/pkg/models"
)

// NullStringToString converts sql.NullString to string
func NullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// StringToNullString converts string to sql.NullString
func StringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullInt64ToInt converts sql.NullInt64 to int
func NullInt64ToInt(ni sql.NullInt64) int {
	if ni.Valid {
		return int(ni.Int64)
	}
	return 0
}

// IntToNullInt64 converts int to sql.NullInt64
func IntToNullInt64(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

// ConvertDBBookToModel converts database Book to models.Book
func ConvertDBBookToModel(dbBook GetBookRow, authors []string, categories []string, shelfName string) models.Book {
	return models.Book{
		ISBN:          int(dbBook.Isbn),
		Title:         NullStringToString(dbBook.Title),
		Description:   NullStringToString(dbBook.Description),
		Publisher:     NullStringToString(dbBook.Publisher),
		PublishedDate: NullStringToString(dbBook.PublishedDate),
		Pages:         NullInt64ToInt(dbBook.Pages),
		Language:      NullStringToString(dbBook.Language),
		CoverURL:      NullStringToString(dbBook.CoverUrl),
		RowNumber:     NullInt64ToInt(dbBook.RowNumber),
		ShelfID:       NullInt64ToInt(dbBook.ShelfID),
		ShelfName:     shelfName,
		Authors:       authors,
		Categories:    categories,
	}
}

// ConvertDBBookRowToModel converts GetAllBooksRow to models.Book
func ConvertDBBookRowToModel(dbBook GetAllBooksRow, authors []string, categories []string, shelfName string) models.Book {
	return models.Book{
		ISBN:          int(dbBook.Isbn),
		Title:         NullStringToString(dbBook.Title),
		Description:   NullStringToString(dbBook.Description),
		Publisher:     NullStringToString(dbBook.Publisher),
		PublishedDate: NullStringToString(dbBook.PublishedDate),
		Pages:         NullInt64ToInt(dbBook.Pages),
		Language:      NullStringToString(dbBook.Language),
		CoverURL:      NullStringToString(dbBook.CoverUrl),
		RowNumber:     NullInt64ToInt(dbBook.RowNumber),
		ShelfID:       NullInt64ToInt(dbBook.ShelfID),
		ShelfName:     shelfName,
		Authors:       authors,
		Categories:    categories,
	}
}

// ConvertNullStringSliceToStringSlice converts []sql.NullString to []string
func ConvertNullStringSliceToStringSlice(nullStrings []sql.NullString) []string {
	result := make([]string, 0, len(nullStrings))
	for _, ns := range nullStrings {
		if ns.Valid {
			result = append(result, ns.String)
		}
	}
	return result
}