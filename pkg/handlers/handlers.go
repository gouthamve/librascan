package handlers

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

	"github.com/gouthamve/librascan/pkg/models"
)

type Librascan struct {
	db *sql.DB
}

func NewLibrascan(db *sql.DB) *Librascan {
	return &Librascan{db: db}
}

// LookupBookHandler handles requests for a book lookup by ISBN using Open Library API.
func (ls *Librascan) LookupBookHandler(c echo.Context) error {
	isbnStr := c.Param("isbn")
	if isbnStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbnStr = strings.ReplaceAll(isbnStr, "-", "") // Remove any hyphens from ISBN
	isbn, err := strconv.Atoi(isbnStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	gb := models.GoogleBook{}
	ol := models.OpenLibraryBook{}

	googleBookResp, err := getBookFromGoogleBooks(isbnStr)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	} else {
		slog.Error("failed to fetch from Google Books API", "error", err, "isbn", isbnStr)
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(isbnStr)
	if err == nil && len(*openLibraryBookResp) > 0 {
		ol = (*openLibraryBookResp)[fmt.Sprintf("ISBN:%s", isbnStr)]
	} else {
		slog.Error("failed to fetch from Open Library API", "error", err, "isbn", isbnStr)
	}

	book := createBookFromAPIData(gb, ol)
	book.ISBN = isbn

	return c.JSONPretty(http.StatusOK, models.DebugResponse{
		Book:                book,
		GoogleBooksResponse: googleBookResp,
		OpenLibraryResponse: openLibraryBookResp,
	}, "  ")
}

// AddBookFromISBN handles requests to add a book to the database by looking up its ISBN.
func (ls *Librascan) AddBookFromISBN(c echo.Context) error {
	isbnStr := c.Param("isbn")
	if isbnStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbnStr = strings.ReplaceAll(isbnStr, "-", "")
	if len(isbnStr) != 13 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}
	isbn, err := strconv.Atoi(isbnStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	// Read and validate query parameters.
	shelfID := 0
	rowNumber := 0

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

	gb := models.GoogleBook{}
	ol := models.OpenLibraryBook{}

	googleBookResp, err := getBookFromGoogleBooks(isbnStr)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(isbnStr)
	if err == nil && len(*openLibraryBookResp) > 0 {
		ol = (*openLibraryBookResp)[fmt.Sprintf("ISBN:%s", isbnStr)]
	}

	book := createBookFromAPIData(gb, ol)
	book.ISBN = isbn
	book.RowNumber = rowNumber
	book.ShelfID = shelfID

	if err := storeBook(ls.db, book); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, book)
}

// GetBookByISBN handles fetching a book from the database by ISBN.
func (ls *Librascan) GetBookByISBN(c echo.Context) error {
	isbnStr := c.Param("isbn")
	if isbnStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbnStr = strings.ReplaceAll(isbnStr, "-", "")
	if len(isbnStr) != 13 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	isbn, err := strconv.Atoi(isbnStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	var book models.Book
	book, err = getBook(ls.db, isbn)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
	}

	return c.JSON(http.StatusOK, book)
}

func (ls *Librascan) GetAllBooks(c echo.Context) error {
	rows, err := ls.db.Query("SELECT isbn, title, description, publisher, published_date, pages, language, cover_url, shelf_id, row_number FROM books;")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		book := models.Book{}

		if err := rows.Scan(&book.ISBN, &book.Title, &book.Description, &book.Publisher, &book.PublishedDate, &book.Pages, &book.Language, &book.CoverURL, &book.ShelfID, &book.RowNumber); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "scan error: " + err.Error()})
		}

		authors, err := getAuthors(ls.db, book.ISBN)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "get authors error: " + err.Error()})
		}
		book.Authors = authors

		categories, err := getCategories(ls.db, book.ISBN)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "get categories error: " + err.Error()})
		}
		book.Categories = categories

		shelf, err := getShelf(ls.db, book.ShelfID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "get shelf error: " + err.Error()})
		}
		book.ShelfName = shelf.Name

		books = append(books, book)
	}

	return c.JSON(http.StatusOK, books)
}

// Add a new handler to lookup shelf name by id.
func (ls *Librascan) LookupShelfNameHandler(c echo.Context) error {
	shelfIDStr := c.Param("id")
	if shelfIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Shelf id is required"})
	}
	shelfID, err := strconv.Atoi(shelfIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid shelf id"})
	}

	shelf, err := getShelf(ls.db, shelfID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "shelf not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	return c.JSON(http.StatusOK, shelf)
}

// DeleteBookByISBN handles deletion of a book from the database by ISBN.
func (ls *Librascan) DeleteBookByISBN(c echo.Context) error {
	isbnStr := c.Param("isbn")
	if isbnStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbnStr = strings.ReplaceAll(isbnStr, "-", "")
	if len(isbnStr) != 13 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	isbn, err := strconv.Atoi(isbnStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ISBN"})
	}

	rows, err := deleteBook(ls.db, isbn)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if rows == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
	}

	return c.NoContent(http.StatusNoContent)
}

func deleteBook(db *sql.DB, isbn int) (int64, error) {
	result, err := db.Exec("DELETE FROM books WHERE isbn = ?", isbn)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func storeBook(db *sql.DB, book models.Book) error {
	_, err := db.Exec(`
		INSERT INTO books 
		(isbn, title, description, publisher, published_date, pages, language, cover_url, row_number, shelf_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(isbn) DO UPDATE SET
			row_number = excluded.row_number,
			shelf_id = excluded.shelf_id`,
		book.ISBN, book.Title, book.Description, book.Publisher, book.PublishedDate,
		book.Pages, book.Language, book.CoverURL, book.RowNumber, book.ShelfID)
	if err != nil {
		return err
	}
	for _, author := range book.Authors {
		_, err = db.Exec("INSERT OR IGNORE INTO authors (isbn, name) VALUES (?, ?)", book.ISBN, author)
		if err != nil {
			return err
		}
	}

	for _, category := range book.Categories {
		_, err = db.Exec("INSERT OR IGNORE INTO categories (isbn, name) VALUES (?, ?)", book.ISBN, category)
		if err != nil {
			return err
		}
	}

	return nil
}

func getBook(db *sql.DB, isbn int) (models.Book, error) {
	var book models.Book
	row := db.QueryRow("SELECT isbn, title, description, publisher, published_date, pages, language, cover_url, row_number, shelf_id FROM books WHERE isbn = ?", isbn)
	err := row.Scan(&book.ISBN, &book.Title, &book.Description, &book.Publisher, &book.PublishedDate, &book.Pages, &book.Language, &book.CoverURL, &book.RowNumber, &book.ShelfID)
	if err != nil {
		return models.Book{}, err
	}

	authors, err := getAuthors(db, isbn)
	if err != nil {
		return models.Book{}, err
	}
	book.Authors = authors

	categories, err := getCategories(db, isbn)
	if err != nil {
		return models.Book{}, err
	}
	book.Categories = categories

	row = db.QueryRow("SELECT name FROM shelfs WHERE id = ?", book.ShelfID)
	err = row.Scan(&book.ShelfName)
	if err != nil {
		return book, err
	}

	return book, nil
}

func getAuthors(db *sql.DB, isbn int) ([]string, error) {
	rows, err := db.Query("SELECT name FROM authors WHERE isbn = ?", isbn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var authors []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		authors = append(authors, name)
	}
	return authors, rows.Err()
}

func getCategories(db *sql.DB, isbn int) ([]string, error) {
	rows, err := db.Query("SELECT name FROM categories WHERE isbn = ?", isbn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		categories = append(categories, name)
	}
	return categories, rows.Err()
}

func getShelf(db *sql.DB, shelfID int) (models.Shelf, error) {
	shelf := models.Shelf{
		ID: shelfID,
	}
	err := db.QueryRow("SELECT name, rows_count FROM shelfs WHERE id = ?", shelfID).Scan(&shelf.Name, &shelf.RowCount)
	if err != nil {
		return shelf, err
	}

	return shelf, nil
}

func createBookFromAPIData(gb models.GoogleBook, ol models.OpenLibraryBook) models.Book {
	book := models.Book{}

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
		if len(ol.Publishers) > 0 {
			book.Publisher = ol.Publishers[0].Name
		}
		book.PublishedDate = ol.PublishDate
	}

	return book
}

func getBookFromGoogleBooks(isbn string) (*models.GoogleBooksResponse, error) {
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

	var response models.GoogleBooksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response")
	}

	if response.TotalItems == 0 {
		return nil, fmt.Errorf("book not found")
	}

	return &response, nil
}

func getBookFromOpenLibrary(isbn string) (*models.OpenLibraryResponse, error) {
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

	var response models.OpenLibraryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response")
	}

	return &response, nil
}
