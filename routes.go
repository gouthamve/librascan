package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
