package cron

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/invopop/jsonschema"
)

type PerplexityJob struct {
	db     *sql.DB
	apiKey string
}

func NewPerplexityJob(db *sql.DB, apiKey string) *PerplexityJob {
	return &PerplexityJob{db: db, apiKey: apiKey}
}

func (p *PerplexityJob) Name() string {
	return "perplexity_enricher"
}

func (p *PerplexityJob) Period() time.Duration {
	return 10 * time.Minute
}

func (p *PerplexityJob) Run() error {
	rows, err := p.db.Query("SELECT isbn FROM books WHERE is_ai_enriched = 0")
	if err != nil {
		return fmt.Errorf("failed to query books: %v", err)
	}

	unenrichedISBNs := []int{}

	for rows.Next() {
		var isbn int
		if err := rows.Scan(&isbn); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		unenrichedISBNs = append(unenrichedISBNs, isbn)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %v", err)
	}

	if err := rows.Close(); err != nil {
		return fmt.Errorf("rows close error: %v", err)
	}

	for _, isbn := range unenrichedISBNs {
		if err := p.enrichBook(isbn); err != nil {
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	var result PPLXResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("response JSON decode error: %v; isbn: %d", err, isbn)
	}

	var book Book
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &book); err != nil {
		return fmt.Errorf("book JSON unmarshal error: %v", err)
	}

	// Update title if none exists.
	_, err = p.db.Exec("UPDATE books SET title = ? WHERE isbn = ? AND (title IS NULL or title = '')", book.Title, isbn)
	if err != nil {
		return fmt.Errorf("failed to update title: %v", err)
	}

	// Update description if none exists.
	_, err = p.db.Exec("UPDATE books SET description = ? WHERE isbn = ? AND (description IS NULL or description = '')", book.Description, isbn)
	if err != nil {
		return fmt.Errorf("failed to update description: %v", err)
	}

	// Update publish_date if none exists.
	_, err = p.db.Exec("UPDATE books SET published_date = ? WHERE isbn = ? AND (published_date IS NULL or published_date = '')", book.PublishDate, isbn)
	if err != nil {
		return fmt.Errorf("failed to update published_date: %v", err)
	}

	for _, author := range book.Authors {
		_, err = p.db.Exec("INSERT OR IGNORE INTO authors (name, isbn) VALUES (?, ?)", author, isbn)
		if err != nil {
			return fmt.Errorf("failed to insert author: %v", err)
		}
	}

	for _, genre := range book.Genres {
		_, err = p.db.Exec("INSERT OR IGNORE INTO categories (name, isbn) VALUES (?, ?)", genre, isbn)
		if err != nil {
			return fmt.Errorf("failed to insert genre: %v", err)
		}
	}

	_, err = p.db.Exec("UPDATE books SET is_ai_enriched = 1 WHERE isbn = ?", isbn)
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
