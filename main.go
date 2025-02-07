package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var requestsTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "requests_total",
		Help: "Total number of requests received",
	},
)

func init() {
	// Register the custom metric
	prometheus.MustRegister(requestsTotal)
}

func main() {
	// Initialize the Echo instance
	e := echo.New()

	// Setup routes in routes.go
	SetupRoutes(e, requestsTotal)

	log.Println("Starting server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
