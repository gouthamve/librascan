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

func TestBookRESTEndpoints(t *testing.T) {
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

	// Test 3: Delete the book
	deleteReq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/books/9783836526722", ts.URL), nil)
	if err != nil {
		t.Fatalf("failed to create DELETE request: %v", err)
	}

	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatalf("failed to make DELETE request: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		t.Fatalf("expected status 204 for DELETE, got %d, body: %s", deleteResp.StatusCode, string(body))
	}

	// Test 4: Verify the book is deleted by trying to GET it again
	getDeletedResp, err := http.Get(fmt.Sprintf("%s/books/9783836526722", ts.URL))
	if err != nil {
		t.Fatalf("failed to make GET request after delete: %v", err)
	}
	defer getDeletedResp.Body.Close()

	if getDeletedResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(getDeletedResp.Body)
		t.Fatalf("expected status 404 after delete, got %d, body: %s", getDeletedResp.StatusCode, string(body))
	}
}

func TestLookupShelfNameHandler(t *testing.T) {
	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Test 1: Lookup the default "unknown" shelf (ID 0)
	resp, err := http.Get(fmt.Sprintf("%s/shelf/0", ts.URL))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d, body: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// Parse response
	var shelf models.Shelf
	if err := json.Unmarshal(body, &shelf); err != nil {
		t.Fatalf("failed to unmarshal shelf response: %v", err)
	}

	// Verify the default shelf
	if shelf.ID != 0 {
		t.Errorf("expected shelf ID 0, got %d", shelf.ID)
	}
	if shelf.Name != "unknown" {
		t.Errorf("expected shelf name 'unknown', got '%s'", shelf.Name)
	}
	if shelf.RowCount != 0 {
		t.Errorf("expected row count 0, got %d", shelf.RowCount)
	}

	// Test 2: Lookup shelf ID 2 (office-small)
	resp2, err := http.Get(fmt.Sprintf("%s/shelf/2", ts.URL))
	if err != nil {
		t.Fatalf("failed to make request for shelf 2: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected status 200 for shelf 2, got %d, body: %s", resp2.StatusCode, string(body))
	}

	// Read response
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("failed to read response body for shelf 2: %v", err)
	}

	// Parse response
	var shelf2 models.Shelf
	if err := json.Unmarshal(body2, &shelf2); err != nil {
		t.Fatalf("failed to unmarshal shelf 2 response: %v", err)
	}

	// Verify shelf 2
	if shelf2.ID != 2 {
		t.Errorf("expected shelf ID 2, got %d", shelf2.ID)
	}
	if shelf2.Name != "office-small" {
		t.Errorf("expected shelf name 'office-small', got '%s'", shelf2.Name)
	}
	if shelf2.RowCount != 7 {
		t.Errorf("expected row count 7, got %d", shelf2.RowCount)
	}

	// Test 3: Lookup a non-existent shelf
	resp3, err := http.Get(fmt.Sprintf("%s/shelf/999", ts.URL))
	if err != nil {
		t.Fatalf("failed to make request for non-existent shelf: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("expected status 404 for non-existent shelf, got %d, body: %s", resp3.StatusCode, string(body))
	}

	// Test 4: Invalid shelf ID
	resp4, err := http.Get(fmt.Sprintf("%s/shelf/invalid", ts.URL))
	if err != nil {
		t.Fatalf("failed to make request with invalid shelf ID: %v", err)
	}
	defer resp4.Body.Close()

	if resp4.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp4.Body)
		t.Fatalf("expected status 400 for invalid shelf ID, got %d, body: %s", resp4.StatusCode, string(body))
	}
}

func TestBorrowBookByISBN(t *testing.T) {
	cleanupMocks := setupMockServers(t)
	defer cleanupMocks()

	ts, _, cleanup := setupTestServer(t)
	defer cleanup()

	// First, insert a book to borrow
	resp, err := http.Post(fmt.Sprintf("%s/books/9783836526722", ts.URL), "application/json", nil)
	if err != nil {
		t.Fatalf("failed to make POST request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 for book creation, got %d", resp.StatusCode)
	}

	// Test 1: Borrow the book with a new person
	borrowReq := models.BorrowRequest{
		ISBN:       9783836526722,
		PersonName: "John Doe",
	}
	borrowJSON, _ := json.Marshal(borrowReq)
	
	borrowResp, err := http.Post(fmt.Sprintf("%s/books/borrow", ts.URL), "application/json", bytes.NewReader(borrowJSON))
	if err != nil {
		t.Fatalf("failed to make borrow request: %v", err)
	}
	defer borrowResp.Body.Close()

	if borrowResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(borrowResp.Body)
		t.Fatalf("expected status 204 for borrow request, got %d, body: %s", borrowResp.StatusCode, string(body))
	}

	// Test 2: Try to borrow a non-existent book
	nonExistentBorrowReq := models.BorrowRequest{
		ISBN:       1234567890123,
		PersonName: "Jane Doe",
	}
	nonExistentJSON, _ := json.Marshal(nonExistentBorrowReq)
	
	nonExistentResp, err := http.Post(fmt.Sprintf("%s/books/borrow", ts.URL), "application/json", bytes.NewReader(nonExistentJSON))
	if err != nil {
		t.Fatalf("failed to make borrow request for non-existent book: %v", err)
	}
	defer nonExistentResp.Body.Close()

	if nonExistentResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(nonExistentResp.Body)
		t.Fatalf("expected status 404 for non-existent book borrow, got %d, body: %s", nonExistentResp.StatusCode, string(body))
	}

	// Test 3: Borrow with invalid JSON
	invalidResp, err := http.Post(fmt.Sprintf("%s/books/borrow", ts.URL), "application/json", bytes.NewReader([]byte("invalid json")))
	if err != nil {
		t.Fatalf("failed to make borrow request with invalid JSON: %v", err)
	}
	defer invalidResp.Body.Close()

	if invalidResp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(invalidResp.Body)
		t.Fatalf("expected status 400 for invalid JSON, got %d, body: %s", invalidResp.StatusCode, string(body))
	}

	// Test 4: Verify the person was created
	peopleResp, err := http.Get(fmt.Sprintf("%s/people", ts.URL))
	if err != nil {
		t.Fatalf("failed to get people: %v", err)
	}
	defer peopleResp.Body.Close()

	if peopleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for get people, got %d", peopleResp.StatusCode)
	}

	var people []models.Person
	body, _ := io.ReadAll(peopleResp.Body)
	if err := json.Unmarshal(body, &people); err != nil {
		t.Fatalf("failed to unmarshal people response: %v", err)
	}

	// Verify that John Doe was created
	found := false
	for _, person := range people {
		if person.Name == "John Doe" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find John Doe in people list, but didn't")
	}
}
