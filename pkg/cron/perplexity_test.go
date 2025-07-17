package cron

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gouthamve/librascan/migrations"
	_ "modernc.org/sqlite"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ctx := t.Context()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Run necessary migrations
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

	return db
}

func TestPerplexityJob_Run_Success(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	// Insert a test book that needs enrichment
	_, err := db.Exec(`INSERT INTO books (isbn, title, is_ai_enriched) VALUES (?, ?, ?)`, 
		9783836526722, "Test Book", 0)
	if err != nil {
		t.Fatalf("failed to insert test book: %v", err)
	}

	// Create mock response
	mockResponse := PPLXResponse{
		ID:    "test-id",
		Model: "sonar",
		Choices: []struct {
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Delta struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"delta"`
		}{
			{
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role: "assistant",
					Content: `{
						"title": "The Fairy Tales of the Brothers Grimm",
						"description": "A collection of classic fairy tales",
						"authors": ["Wilhelm Grimm", "Jacob Grimm"],
						"publish_date": "2011",
						"genres": ["Fairy Tales", "Classics"]
					}`,
				},
			},
		},
	}

	responseBody, _ := json.Marshal(mockResponse)

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify request
			if req.Method != "POST" {
				t.Errorf("expected POST method, got %s", req.Method)
			}
			if req.URL.String() != "https://api.perplexity.ai/chat/completions" {
				t.Errorf("unexpected URL: %s", req.URL.String())
			}
			if req.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("unexpected Authorization header: %s", req.Header.Get("Authorization"))
			}

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
			}, nil
		},
	}

	job := NewPerplexityJob(db, "test-api-key")
	job.httpClient = mockClient

	// Run the job
	err = job.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify the book was enriched
	var isEnriched int
	var description string
	err = db.QueryRow("SELECT is_ai_enriched, description FROM books WHERE isbn = ?", 9783836526722).
		Scan(&isEnriched, &description)
	if err != nil {
		t.Fatalf("failed to query book: %v", err)
	}

	if isEnriched != 1 {
		t.Errorf("expected is_ai_enriched to be 1, got %d", isEnriched)
	}

	if description != "A collection of classic fairy tales" {
		t.Errorf("expected description to be updated, got %s", description)
	}

	// Verify authors were inserted
	var authorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM authors WHERE isbn = ?", 9783836526722).Scan(&authorCount)
	if err != nil {
		t.Fatalf("failed to count authors: %v", err)
	}
	if authorCount != 2 {
		t.Errorf("expected 2 authors, got %d", authorCount)
	}

	// Verify categories were inserted
	var categoryCount int
	err = db.QueryRow("SELECT COUNT(*) FROM categories WHERE isbn = ?", 9783836526722).Scan(&categoryCount)
	if err != nil {
		t.Fatalf("failed to count categories: %v", err)
	}
	if categoryCount != 2 {
		t.Errorf("expected 2 categories, got %d", categoryCount)
	}
}

func TestPerplexityJob_Run_APIError(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	// Insert a test book
	_, err := db.Exec(`INSERT INTO books (isbn, title, is_ai_enriched) VALUES (?, ?, ?)`, 
		9783836526722, "Test Book", 0)
	if err != nil {
		t.Fatalf("failed to insert test book: %v", err)
	}

	// Create mock HTTP client that returns an error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewReader([]byte("Internal Server Error"))),
			}, nil
		},
	}

	job := NewPerplexityJob(db, "test-api-key")
	job.httpClient = mockClient

	// Run should fail
	err = job.Run()
	if err == nil {
		t.Error("expected Run() to fail with API error")
	}

	// Verify the book was not enriched
	var isEnriched int
	err = db.QueryRow("SELECT is_ai_enriched FROM books WHERE isbn = ?", 9783836526722).
		Scan(&isEnriched)
	if err != nil {
		t.Fatalf("failed to query book: %v", err)
	}

	if isEnriched != 0 {
		t.Errorf("expected is_ai_enriched to remain 0, got %d", isEnriched)
	}
}

func TestPerplexityJob_Name(t *testing.T) {
	job := &PerplexityJob{}
	if job.Name() != "perplexity_enricher" {
		t.Errorf("expected name 'perplexity_enricher', got %s", job.Name())
	}
}

func TestPerplexityJob_Period(t *testing.T) {
	job := &PerplexityJob{}
	expected := 10 * time.Minute
	if job.Period() != expected {
		t.Errorf("expected period %v, got %v", expected, job.Period())
	}
}

func TestPerplexityJob_Run_NoBooks(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	// No books to enrich
	job := NewPerplexityJob(db, "test-api-key")
	job.httpClient = &MockHTTPClient{}

	// Run should succeed with no books
	err := job.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
}

func TestPerplexityJob_Run_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	// Insert a test book
	_, err := db.Exec(`INSERT INTO books (isbn, title, is_ai_enriched) VALUES (?, ?, ?)`, 
		9783836526722, "Test Book", 0)
	if err != nil {
		t.Fatalf("failed to insert test book: %v", err)
	}

	// Create mock HTTP client that returns invalid JSON
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
			}, nil
		},
	}

	job := NewPerplexityJob(db, "test-api-key")
	job.httpClient = mockClient

	// Run should fail
	err = job.Run()
	if err == nil {
		t.Error("expected Run() to fail with invalid JSON")
	}
}