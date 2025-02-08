package nococlient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http" // added for DELETE request
	"net/url"
)

type NocoClient struct {
	URL    string
	APIKey string
}

func NewNocoClient(rawURL, apiKey string) (*NocoClient, error) {
	// validate the URL
	_, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	return &NocoClient{
		URL:    rawURL,
		APIKey: apiKey,
	}, nil
}

func (c *NocoClient) ListBases() error {
	bases, err := c.listBases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	for _, base := range bases.List {
		fmt.Printf("ID: %s, Title: %s\n", base.ID, base.Title)
	}

	// Implement the list databases API
	return nil
}

func (c *NocoClient) CreateBaseIfNotExists(name string) error {
	bases, err := c.listBases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	for _, base := range bases.List {
		if base.Title == name {
			slog.Info("Database already exists", "name", name)
			return nil
		}
	}

	path := "/api/v2/meta/bases"
	fullURL, err := url.JoinPath(c.URL, path)
	if err != nil {
		return fmt.Errorf("failed to join URL: %w", err)
	}

	base := Base{
		Title: name,
	}
	base.Sources = append(base.Sources, Source{
		ID: "bn6r8lznm2dlx1m",
		// IsMeta: true,
		Type: "sqlite3",
	})

	data, err := json.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal base: %w", err)
	}

	client := NewHTTPClient(c.APIKey)
	resp, err := client.Post(fullURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to create database. Status Code: %s, Body: %s", resp.Status, body)
	}

	return nil
}

func (c *NocoClient) DeleteBase(name string) error {
	bases, err := c.listBases()
	if err != nil {
		return fmt.Errorf("failed to list bases: %w", err)
	}

	var baseID string
	for _, base := range bases.List {
		if base.Title == name {
			baseID = base.ID
			break
		}
	}
	if baseID == "" {
		return fmt.Errorf("base %s not found", name)
	}

	path := fmt.Sprintf("/api/v2/meta/bases/%s", baseID)
	fullURL, err := url.JoinPath(c.URL, path)
	if err != nil {
		return fmt.Errorf("failed to join URL: %w", err)
	}

	client := NewHTTPClient(c.APIKey)
	req, err := http.NewRequest("DELETE", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete base: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete base. Status: %s, Body: %s", resp.Status, body)
	}

	return nil
}

func (c *NocoClient) listBases() (*BaseList, error) {
	path := "/api/v2/meta/bases"
	fullURL, err := url.JoinPath(c.URL, path)
	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}

	client := NewHTTPClient(c.APIKey)

	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer resp.Body.Close()

	bases := BaseList{}
	if err := json.NewDecoder(resp.Body).Decode(&bases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bases, nil
}
