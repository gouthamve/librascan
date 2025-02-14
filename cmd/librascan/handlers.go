package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/labstack/echo/v4"
)

type librascan struct {
	db *sql.DB
}

func newLibrascan(db *sql.DB) *librascan {
	return &librascan{db: db}
}

// LookupBookHandler handles requests for a book lookup by ISBN using Open Library API.
func (ls *librascan) LookupBookHandler(c echo.Context) error {
	isbn := c.Param("isbn")
	if isbn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbn = strings.ReplaceAll(isbn, "-", "") // Remove any hyphens from ISBN

	gb := GoogleBook{}
	ol := OpenLibraryBook{}

	googleBookResp, err := getBookFromGoogleBooks(isbn)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	} else {
		slog.Error("failed to fetch from Google Books API", "error", err, "isbn", isbn)
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(isbn)
	if err == nil && len(*openLibraryBookResp) > 0 {
		ol = (*openLibraryBookResp)[fmt.Sprintf("ISBN:%s", isbn)]
	} else {
		slog.Error("failed to fetch from Open Library API", "error", err, "isbn", isbn)
	}

	book := createBookFromAPIData(gb, ol)
	book.ISBN = isbn

	return c.JSONPretty(http.StatusOK, DebugResponse{
		Book:                book,
		GoogleBooksResponse: googleBookResp,
		OpenLibraryResponse: openLibraryBookResp,
	}, "  ")
}

// AddBookFromISBN handles requests to add a book to the database by looking up its ISBN.
func (ls *librascan) AddBookFromISBN(c echo.Context) error {
	isbn := c.Param("isbn")
	if isbn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbn = strings.ReplaceAll(isbn, "-", "")

	// Read and validate query parameters.
	shelfID := 0
	rowNumber := 0
	var err error

	rowNumberStr := c.QueryParam("row_number")
	shelfIDStr := c.QueryParam("shelf_id")
	if rowNumberStr != "" {
		rowNumber, err = strconv.Atoi(rowNumberStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid row_number"})
		}
	}
	if shelfIDStr != "" {
		shelfID, err = strconv.Atoi(shelfIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid shelf_id"})
		}
	}

	gb := GoogleBook{}
	ol := OpenLibraryBook{}

	googleBookResp, err := getBookFromGoogleBooks(isbn)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(isbn)
	if err == nil && len(*openLibraryBookResp) > 0 {
		ol = (*openLibraryBookResp)[fmt.Sprintf("ISBN:%s", isbn)]
	}

	book := createBookFromAPIData(gb, ol)
	book.ISBN = isbn

	// Updated INSERT statement with row_id and shelf_id.
	_, err = ls.db.Exec(`
		INSERT INTO books 
		(isbn, title, description, authors, categories, publisher, published_date, pages, language, cover_url, row_number, shelf_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(isbn) DO UPDATE SET
			row_number = excluded.row_number,
			shelf_id = excluded.shelf_id`,
		book.ISBN, book.Title, book.Description, fmt.Sprintf("jsonb_array(%s)", strings.Join(book.Authors, ",")),
		fmt.Sprintf("jsonb_array(%s)", strings.Join(book.Categories, ",")), book.Publisher, book.PublishedDate,
		book.Pages, book.Language, book.CoverURL, rowNumber, shelfID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	for _, author := range book.Authors {
		_, err = ls.db.Exec("INSERT OR IGNORE INTO authors (isbn, name) VALUES (?, ?)", book.ISBN, author)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	for _, category := range book.Categories {
		_, err = ls.db.Exec("INSERT OR IGNORE INTO categories (isbn, name) VALUES (?, ?)", book.ISBN, category)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusCreated, book)
}

func createBookFromAPIData(gb GoogleBook, ol OpenLibraryBook) Book {
	book := Book{}

	if gb.ID != "" {
		book.Title = gb.VolumeInfo.Title
		book.Description = gb.VolumeInfo.Description
		book.Authors = gb.VolumeInfo.Authors
		book.Categories = gb.VolumeInfo.Categories
		book.Publisher = gb.VolumeInfo.Publisher
		book.PublishedDate = gb.VolumeInfo.PublishedDate
		book.Pages = gb.VolumeInfo.PageCount
		book.Language = gb.VolumeInfo.Language
		book.CoverURL = gb.VolumeInfo.ImageLinks.Thumbnail
	}

	if ol.Key != "" {
		largeCoverURL := ol.Cover.Large
		if largeCoverURL != "" {
			book.CoverURL = largeCoverURL
		}

		if book.Title != "" {
			return book
		}
		book.Title = ol.Title
		book.Authors = []string{}
		for _, author := range ol.Authors {
			book.Authors = append(book.Authors, author.Name)
		}
		book.Publisher = ol.Publishers[0].Name
		book.PublishedDate = ol.PublishDate
	}

	return book
}

func getBookFromGoogleBooks(isbn string) (*GoogleBooksResponse, error) {
	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", isbn)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching data from API")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response GoogleBooksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response")
	}

	if response.TotalItems == 0 {
		return nil, fmt.Errorf("book not found")
	}

	return &response, nil
}

func getBookFromOpenLibrary(isbn string) (*OpenLibraryResponse, error) {
	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching data")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 || string(body) == "{}" {
		return nil, fmt.Errorf("book not found")
	}

	var response OpenLibraryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response")
	}

	return &response, nil
}

// Add a new handler to lookup shelf name by id.
func (ls *librascan) LookupShelfNameHandler(c echo.Context) error {
	shelfIDStr := c.Param("id")
	if shelfIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Shelf id is required"})
	}
	shelfID, err := strconv.Atoi(shelfIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid shelf id"})
	}

	var shelfName string
	var rowCount int
	err = ls.db.QueryRow("SELECT name, rows_count FROM shelfs WHERE id = ?", shelfID).Scan(&shelfName, &rowCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "shelf not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	shelf := Shelf{
		ID:       shelfID,
		Name:     shelfName,
		RowCount: rowCount,
	}

	return c.JSON(http.StatusOK, shelf)
}
