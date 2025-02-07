package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// lookupBookHandler handles requests for a book lookup by ISBN using Open Library API.
func lookupBookHandler(c echo.Context) error {
	isbn := c.Param("isbn")
	if isbn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ISBN is required"})
	}

	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	resp, err := http.Get(url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "error fetching data"})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSONBlob(http.StatusOK, body)
}

// SetupRoutes registers HTTP endpoints using the Echo instance.
func SetupRoutes(e *echo.Echo, requestsTotal prometheus.Counter) {
	// Root endpoint that increments the metric
	e.GET("/", func(c echo.Context) error {
		requestsTotal.Inc()
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Prometheus metrics endpoint wrapped for Echo
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// New endpoint to lookup a book by ISBN.
	e.GET("/lookup/:isbn", lookupBookHandler)
}
