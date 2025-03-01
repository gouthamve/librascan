package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gouthamve/librascan/pkg/cron"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
)

// serve runs migrations, starts the HTTP server and routes.
func serve(pplxAPIKey string) {
	if err := os.MkdirAll("./.db", 0755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

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
	e.Use(echoprometheus.NewMiddleware("librascan"))
	e.GET("/metrics", echoprometheus.NewHandler())

	// Setup routes in routes.go
	SetupRoutes(e, db)

	// Setup cron jobs
	setupCronJobs(db, pplxAPIKey)

	// Start the server
	log.Println("Starting server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}

func setupCronJobs(db *sql.DB, pplxAPIKey string) {
	if pplxAPIKey == "" {
		log.Println("Perplexity API key not set, skipping perplexity enrichment")
		return
	}

	pplxCron := cron.NewPerplexityJob(db, pplxAPIKey)
	cr := cron.NewCronRunner([]cron.Job{pplxCron})

	cr.Run()
}
