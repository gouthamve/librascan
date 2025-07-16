package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gouthamve/librascan/migrations"
	"github.com/gouthamve/librascan/pkg/handlers"
	"github.com/gouthamve/librascan/pkg/models"
	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) (*httptest.Server, *sql.DB, func()) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Run migrations
	ctx := t.Context()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	if err := migrations.Up0001(ctx, tx); err != nil {
		t.Fatalf("failed to run migration 0001: %v", err)
	}
	if err := migrations.Up0002(ctx, tx); err != nil {
		t.Fatalf("failed to run migration 0002: %v", err)
	}
	if err := migrations.Up0003(ctx, tx); err != nil {
		t.Fatalf("failed to run migration 0003: %v", err)
	}
	if err := migrations.Up0004(ctx, tx); err != nil {
		t.Fatalf("failed to run migration 0004: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// Setup Echo server
	e := echo.New()
	SetupRoutes(e, db)

	// Create test server
	ts := httptest.NewServer(e)

	cleanup := func() {
		ts.Close()
		db.Close()
	}

	return ts, db, cleanup
}

func loadTestData(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read test data file %s: %v", filename, err)
	}
	return data
}

func compareJSON(t *testing.T, expected, actual []byte, description string) {
	var expectedJSON, actualJSON interface{}
	
	if err := json.Unmarshal(expected, &expectedJSON); err != nil {
		t.Fatalf("failed to unmarshal expected JSON for %s: %v", description, err)
	}
	
	if err := json.Unmarshal(actual, &actualJSON); err != nil {
		t.Fatalf("failed to unmarshal actual JSON for %s: %v", description, err)
	}
	
	// Pretty print both for comparison
	expectedPretty, _ := json.MarshalIndent(expectedJSON, "", "  ")
	actualPretty, _ := json.MarshalIndent(actualJSON, "", "  ")
	
	if !bytes.Equal(expectedPretty, actualPretty) {
		t.Errorf("%s mismatch:\nExpected:\n%s\nActual:\n%s", description, string(expectedPretty), string(actualPretty))
	}
}

// Mock servers for external APIs
var mockGoogleBooksServer *httptest.Server
var mockOpenLibraryServer *httptest.Server

func setupMockServers(t *testing.T) func() {
	// Load test data
	lookupData := loadTestData(t, "../../testdata/9783836526722-lookup.json")
	var debugResp models.DebugResponse
	if err := json.Unmarshal(lookupData, &debugResp); err != nil {
		t.Fatalf("failed to unmarshal lookup test data: %v", err)
	}

	// Mock Google Books API
	mockGoogleBooksServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/books/v1/volumes" && r.URL.Query().Get("q") == "isbn:9783836526722" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(debugResp.GoogleBooksResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Mock Open Library API
	mockOpenLibraryServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" && r.URL.Query().Get("bibkeys") == "ISBN:9783836526722" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(debugResp.OpenLibraryResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Override the API URLs in handlers package
	handlers.GoogleBooksAPIURL = mockGoogleBooksServer.URL + "/books/v1/volumes"
	handlers.OpenLibraryAPIURL = mockOpenLibraryServer.URL + "/api/books"

	return func() {
		mockGoogleBooksServer.Close()
		mockOpenLibraryServer.Close()
	}
}

func TestDebugLookupEndpoint(t *testing.T) {
	cleanupMocks := setupMockServers(t)
	defer cleanupMocks()

	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Make request to debug/lookup endpoint
	resp, err := http.Get(fmt.Sprintf("%s/debug/lookup/9783836526722", ts.URL))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Load expected data
	expectedData := loadTestData(t, "../../testdata/9783836526722-lookup.json")

	// Compare JSON responses
	compareJSON(t, expectedData, body, "debug lookup response")
}

func TestInsertAndGetBook(t *testing.T) {
	cleanupMocks := setupMockServers(t)
	defer cleanupMocks()

	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Test 1: Insert book
	resp, err := http.Post(fmt.Sprintf("%s/books/9783836526722", ts.URL), "application/json", nil)
	if err != nil {
		t.Fatalf("failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201, got %d, body: %s", resp.StatusCode, string(body))
	}

	// Read response
	insertBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read insert response body: %v", err)
	}

	// Load expected data
	expectedData := loadTestData(t, "../../testdata/9783836526722-book.json")

	// Compare insert response
	compareJSON(t, expectedData, insertBody, "insert book response")

	// Test 2: Get the book we just inserted
	getResp, err := http.Get(fmt.Sprintf("%s/books/9783836526722", ts.URL))
	if err != nil {
		t.Fatalf("failed to make GET request: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Read response
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("failed to read GET response body: %v", err)
	}

	// Compare GET response
	compareJSON(t, expectedData, getBody, "get book response")
}