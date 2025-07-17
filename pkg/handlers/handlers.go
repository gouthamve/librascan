package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/gouthamve/librascan/pkg/db"
	"github.com/gouthamve/librascan/pkg/models"
)

// API URLs that can be overridden for testing
var (
	GoogleBooksAPIURL = "https://www.googleapis.com/books/v1/volumes"
	OpenLibraryAPIURL = "https://openlibrary.org/api/books"
)

// Embed the templates directory
//go:embed templates/*.html
var templateFS embed.FS

// Template variables
var (
	templates *template.Template
)

func init() {
	// Custom template function to join string slices
	funcMap := template.FuncMap{
		"join": strings.Join,
	}
	
	// Parse templates from embedded filesystem
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

type Librascan struct {
	queries *db.Queries
}

func NewLibrascan(database *sql.DB) *Librascan {
	return &Librascan{queries: db.New(database)}
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

	googleBookResp, err := getBookFromGoogleBooks(c.Request().Context(), isbnStr)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	} else {
		slog.Error("failed to fetch from Google Books API", "error", err, "isbn", isbnStr)
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(c.Request().Context(), isbnStr)
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

	googleBookResp, err := getBookFromGoogleBooks(c.Request().Context(), isbnStr)
	if err == nil && googleBookResp.TotalItems > 0 {
		gb = googleBookResp.Items[0]
	}

	openLibraryBookResp, err := getBookFromOpenLibrary(c.Request().Context(), isbnStr)
	if err == nil && len(*openLibraryBookResp) > 0 {
		ol = (*openLibraryBookResp)[fmt.Sprintf("ISBN:%s", isbnStr)]
	}

	book := createBookFromAPIData(gb, ol)
	book.ISBN = isbn
	book.RowNumber = rowNumber
	book.ShelfID = shelfID

	if err := ls.storeBook(c.Request().Context(), book); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch the shelf name
	if book.ShelfID != 0 {
		shelfName, err := ls.queries.GetShelfName(c.Request().Context(), int64(book.ShelfID))
		if err == nil && shelfName.Valid {
			book.ShelfName = shelfName.String
		}
	} else {
		book.ShelfName = "unknown"
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

	book, err := ls.getBook(c.Request().Context(), int64(isbn))
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
	}

	return c.JSON(http.StatusOK, book)
}

func (ls *Librascan) GenerateHTMLHandler(c echo.Context) error {
	ctx := c.Request().Context()
	books, err := getAllBooks(ctx, ls.queries)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	// Create template data
	data := struct {
		Books []models.Book
	}{
		Books: books,
	}

	// Execute template
	var buf bytes.Buffer
	err = templates.ExecuteTemplate(&buf, "books.html", data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "template error: " + err.Error()})
	}

	return c.HTML(http.StatusOK, buf.String())
}

func (ls *Librascan) GetAllBooks(c echo.Context) error {
	ctx := c.Request().Context()
	books, err := getAllBooks(ctx, ls.queries)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	return c.JSON(http.StatusOK, books)
}

// LookupShelfNameHandler gets shelf name by id.
func (ls *Librascan) LookupShelfNameHandler(c echo.Context) error {
	shelfIDStr := c.Param("id")
	if shelfIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Shelf id is required"})
	}
	shelfID, err := strconv.Atoi(shelfIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid shelf id"})
	}

	shelf, err := ls.queries.GetShelf(c.Request().Context(), int64(shelfID))
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "shelf not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	return c.JSON(http.StatusOK, models.Shelf{
		ID:       int(shelf.ID),
		Name:     db.NullStringToString(shelf.Name),
		RowCount: db.NullInt64ToInt(shelf.RowsCount),
	})
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

	rows, err := ls.queries.DeleteBook(c.Request().Context(), int64(isbn))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if rows == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
	}

	return c.NoContent(http.StatusNoContent)
}

// BorrowBookByISBN handles borrowing a book by ISBN.
func (ls *Librascan) BorrowBookByISBN(c echo.Context) error {
	// Pass BorrowRequest from body.
	var req models.BorrowRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	ctx := c.Request().Context()

	// Check if book exists.
	_, err := ls.getBook(ctx, int64(req.ISBN))
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
	}

	// Check if person exists.
	var personID int64
	personID, err = ls.queries.GetPerson(ctx, req.PersonName)
	if err != nil {
		if err == sql.ErrNoRows {
			// Add person if not exists.
			personID, err = ls.queries.InsertPerson(ctx, req.PersonName)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
			}
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
		}
	}

	// Borrow book.
	err = ls.queries.InsertBorrowing(ctx, db.InsertBorrowingParams{
		Isbn:     int64(req.ISBN),
		PersonID: personID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Query error: " + err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (ls *Librascan) GetPeople(c echo.Context) error {
	dbPeople, err := ls.queries.GetAllPeople(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query error: " + err.Error()})
	}

	people := []models.Person{}
	for _, dbPerson := range dbPeople {
		people = append(people, models.Person{
			ID:   int(dbPerson.ID),
			Name: dbPerson.Name,
		})
	}

	return c.JSON(http.StatusOK, people)
}

// storeBook stores a book in the database using sqlc
func (ls *Librascan) storeBook(ctx context.Context, book models.Book) error {
	// Insert or update book
	err := ls.queries.InsertBook(ctx, db.InsertBookParams{
		Isbn:          int64(book.ISBN),
		Title:         db.StringToNullString(book.Title),
		Description:   db.StringToNullString(book.Description),
		Publisher:     db.StringToNullString(book.Publisher),
		PublishedDate: db.StringToNullString(book.PublishedDate),
		Pages:         db.IntToNullInt64(book.Pages),
		Language:      db.StringToNullString(book.Language),
		CoverUrl:      db.StringToNullString(book.CoverURL),
		RowNumber:     db.IntToNullInt64(book.RowNumber),
		ShelfID:       db.IntToNullInt64(book.ShelfID),
	})
	if err != nil {
		return err
	}

	// Insert authors
	for _, author := range book.Authors {
		err = ls.queries.InsertAuthor(ctx, db.InsertAuthorParams{
			Isbn: sql.NullInt64{Int64: int64(book.ISBN), Valid: true},
			Name: sql.NullString{String: author, Valid: true},
		})
		if err != nil {
			return err
		}
	}

	// Insert categories
	for _, category := range book.Categories {
		err = ls.queries.InsertCategory(ctx, db.InsertCategoryParams{
			Isbn: sql.NullInt64{Int64: int64(book.ISBN), Valid: true},
			Name: sql.NullString{String: category, Valid: true},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// getBook retrieves a book from the database using sqlc
func (ls *Librascan) getBook(ctx context.Context, isbn int64) (models.Book, error) {
	dbBook, err := ls.queries.GetBook(ctx, isbn)
	if err != nil {
		return models.Book{}, err
	}

	authors, err := ls.queries.GetAuthors(ctx, sql.NullInt64{Int64: isbn, Valid: true})
	if err != nil {
		return models.Book{}, err
	}

	categories, err := ls.queries.GetCategories(ctx, sql.NullInt64{Int64: isbn, Valid: true})
	if err != nil {
		return models.Book{}, err
	}

	shelfName := "unknown"
	if dbBook.ShelfID.Valid {
		name, err := ls.queries.GetShelfName(ctx, dbBook.ShelfID.Int64)
		if err == nil && name.Valid {
			shelfName = name.String
		}
	}

	return db.ConvertDBBookToModel(
		dbBook,
		db.ConvertNullStringSliceToStringSlice(authors),
		db.ConvertNullStringSliceToStringSlice(categories),
		shelfName,
	), nil
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

func getBookFromGoogleBooks(ctx context.Context, isbn string) (*models.GoogleBooksResponse, error) {
	url := fmt.Sprintf("%s?q=isbn:%s", GoogleBooksAPIURL, isbn)
	resp, err := otelhttp.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

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

func getBookFromOpenLibrary(ctx context.Context, isbn string) (*models.OpenLibraryResponse, error) {
	url := fmt.Sprintf("%s?bibkeys=ISBN:%s&format=json&jscmd=data", OpenLibraryAPIURL, isbn)
	resp, err := otelhttp.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

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

func getAllBooks(ctx context.Context, queries *db.Queries) ([]models.Book, error) {

	dbBooks, err := queries.GetAllBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all books error: %w", err)
	}

	books := []models.Book{}
	for _, dbBook := range dbBooks {
		authors, err := queries.GetAuthors(ctx, sql.NullInt64{Int64: dbBook.Isbn, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("get authors error: %w", err)
		}

		categories, err := queries.GetCategories(ctx, sql.NullInt64{Int64: dbBook.Isbn, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("get categories error: %w", err)
		}

		shelfName := "unknown"
		if dbBook.ShelfID.Valid {
			name, err := queries.GetShelfName(ctx, dbBook.ShelfID.Int64)
			if err == nil && name.Valid {
				shelfName = name.String
			}
		}

		book := db.ConvertDBBookRowToModel(
			dbBook,
			db.ConvertNullStringSliceToStringSlice(authors),
			db.ConvertNullStringSliceToStringSlice(categories),
			shelfName,
		)
		books = append(books, book)
	}

	return books, nil
}

