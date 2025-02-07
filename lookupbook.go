package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// lookupBookHandler handles requests for a book lookup by ISBN using Open Library API.
func lookupBookHandler(c echo.Context) error {
	isbn := c.Param("isbn")
	if isbn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	isbn = strings.ReplaceAll(isbn, "-", "") // Remove any hyphens from ISBN

	googleBook, err := getBookFromGoogleBooks(isbn)
	if err == nil && googleBook.TotalItems > 0 {
		return c.JSONPretty(http.StatusOK, googleBook, "  ")
	}

	slog.Error("failed to fetch from Google Books API", "error", err, "isbn", isbn)

	openLibraryBook, err := getBookFromOpenLibrary(isbn)
	if err == nil && len(*openLibraryBook) > 0 {
		return c.JSONPretty(http.StatusOK, openLibraryBook, "  ")
	} else {
		slog.Error("failed to fetch from Open Library API", "error", err, "isbn", isbn)
	}

	return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
}

func getBookFromGoogleBooks(isbn string) (*GoogleBooksResponse, error) {
	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", isbn)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching data from Google Books API")
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
