package cron

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"
	"context"

	"github.com/invopop/jsonschema"
	"github.com/gouthamve/librascan/pkg/db"
)

// HTTPClient interface for making HTTP requests (allows mocking in tests)
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type PerplexityJob struct {
	queries    *db.Queries
	apiKey     string
	httpClient HTTPClient
}

func NewPerplexityJob(database *sql.DB, apiKey string) *PerplexityJob {
	return &PerplexityJob{
		queries:    db.New(database),
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (p *PerplexityJob) Name() string {
	return "perplexity_enricher"
}

func (p *PerplexityJob) Period() time.Duration {
	return 10 * time.Minute
}

func (p *PerplexityJob) Run() error {
	ctx := context.Background()
	unenrichedISBNs, err := p.queries.GetUnenrichedBooks(ctx)
	if err != nil {
		return fmt.Errorf("failed to query unenriched books: %v", err)
	}

	for _, isbn := range unenrichedISBNs {
		if err := p.enrichBook(int(isbn)); err != nil {
			return fmt.Errorf("enrichment error: %v", err)
		}
	}

	return nil
}

func (p *PerplexityJob) enrichBook(isbn int) error {
	url := "https://api.perplexity.ai/chat/completions"

	promptTmpl := `Tell me about the book with ISBN %d. 
	Please output a JSON object containing the following fields: 
	title, description, authors, publish_date, and genres.`

	prompt := fmt.Sprintf(promptTmpl, isbn)

	// Construct the payload (json_schema placeholder)
	payload := PPLXRequestPayload{
		Model: "sonar",
		Messages: []PPLXMessage{
			{
				Role:    "system",
				Content: "Be precise and concise.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat: PPLXResponseFormat{
			Type: "json_schema",
			JSONSchema: PPLXJSONSchema{
				Schema: toJSONSchema(Book{}),
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("JSON marshal error: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	var result PPLXResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("response JSON decode error: %v; isbn: %d", err, isbn)
	}

	var book Book
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &book); err != nil {
		return fmt.Errorf("book JSON unmarshal error: %v", err)
	}

	ctx := context.Background()

	// Update title if none exists.
	err = p.queries.UpdateBookTitle(ctx, db.UpdateBookTitleParams{
		Title: sql.NullString{String: book.Title, Valid: book.Title != ""},
		Isbn:  int64(isbn),
	})
	if err != nil {
		return fmt.Errorf("failed to update title: %v", err)
	}

	// Update description if none exists.
	err = p.queries.UpdateBookDescription(ctx, db.UpdateBookDescriptionParams{
		Description: sql.NullString{String: book.Description, Valid: book.Description != ""},
		Isbn:        int64(isbn),
	})
	if err != nil {
		return fmt.Errorf("failed to update description: %v", err)
	}

	// Update publish_date if none exists.
	err = p.queries.UpdateBookPublishedDate(ctx, db.UpdateBookPublishedDateParams{
		PublishedDate: sql.NullString{String: book.PublishDate, Valid: book.PublishDate != ""},
		Isbn:          int64(isbn),
	})
	if err != nil {
		return fmt.Errorf("failed to update published_date: %v", err)
	}

	for _, author := range book.Authors {
		err = p.queries.InsertAuthor(ctx, db.InsertAuthorParams{
			Name: sql.NullString{String: author, Valid: true},
			Isbn: sql.NullInt64{Int64: int64(isbn), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to insert author: %v", err)
		}
	}

	for _, genre := range book.Genres {
		err = p.queries.InsertCategory(ctx, db.InsertCategoryParams{
			Name: sql.NullString{String: genre, Valid: true},
			Isbn: sql.NullInt64{Int64: int64(isbn), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to insert genre: %v", err)
		}
	}

	err = p.queries.MarkBookAsEnriched(ctx, int64(isbn))
	if err != nil {
		return fmt.Errorf("failed to update is_ai_enriched: %v", err)
	}

	return nil
}

// Book represents the expected fields.
type Book struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Authors     []string `json:"authors"`
	PublishDate string   `json:"publish_date"`
	Genres      []string `json:"genres"`
}

type PPLXMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PPLXJSONSchema struct {
	Schema *jsonschema.Schema `json:"schema"`
}

type PPLXResponseFormat struct {
	Type       string         `json:"type"`
	JSONSchema PPLXJSONSchema `json:"json_schema"`
	// Regex      struct {
	// 	Regex string `json:"regex"`
	// } `json:"regex,omitempty"`
}

type PPLXResponse struct {
	ID        string   `json:"id"`
	Model     string   `json:"model"`
	Object    string   `json:"object"`
	Created   int      `json:"created"`
	Citations []string `json:"citations"`
	Choices   []struct {
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
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type PPLXRequestPayload struct {
	Model          string             `json:"model"`
	Messages       []PPLXMessage      `json:"messages"`
	ResponseFormat PPLXResponseFormat `json:"response_format"`
}

func toJSONSchema(v any) *jsonschema.Schema {
	schema := jsonschema.Reflect(v)

	return schema.Definitions[reflect.TypeOf(v).Name()]
}
