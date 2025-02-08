package main

import (
	"database/sql"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	_ "modernc.org/sqlite"

	_ "github.com/gouthamve/librascan/migrations"
)

var requestsTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "requests_total",
		Help: "Total number of requests received",
	},
)

var database = "./librascan.db"

func init() {
	// Register the custom metric
	prometheus.MustRegister(requestsTotal)
}

func main() {
	// Run the migrations
	db, err := goose.OpenDBWithDriver("sqlite3", database)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := goose.Up(db, "."); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	if err := db.Close(); err != nil {
		log.Fatalf("failed to close database: %v", err)
	}

	db, err = sql.Open("sqlite", database)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize the Echo instance
	e := echo.New()
	e.Use(middleware.Logger())

	// Setup routes in routes.go
	SetupRoutes(e, requestsTotal, db)

	log.Println("Starting server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
