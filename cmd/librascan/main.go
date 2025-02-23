package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	_ "github.com/gouthamve/librascan/migrations"
	_ "modernc.org/sqlite"

	"github.com/gouthamve/librascan/pkg/readIsbn"
	"github.com/gouthamve/librascan/pkg/tui"
)

var database = "./.db/librascan.db"

// serve runs migrations, starts the HTTP server and routes.
func serve() {
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

	log.Println("Starting server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "librascan",
		Short: "A book lookup server and ISBN CLI tool",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			serve()
		},
	}

	// Add a flag option for server URL in the read-isbn command.
	waitCmd := &cobra.Command{
		Use:   "read-isbn",
		Short: "Start ISBN input loop",
		Run: func(cmd *cobra.Command, args []string) {
			serverURL, err := cmd.Flags().GetString("server-url")
			if err != nil {
				log.Fatalln("cannot get server URL:", err)
			}
			inputDevicePath, err := cmd.Flags().GetString("input-device-path")
			if err != nil {
				log.Fatalln("cannot get inputPath flag:", err)
			}
			readIsbn.StartCLI(serverURL, inputDevicePath)
		},
	}
	waitCmd.Flags().String("server-url", "http://localhost:8080", "Server URL for posting ISBNs.")
	waitCmd.Flags().String("input-device-path", "", "Path to the scanners udev device.")

	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the TUI interface",
		Run: func(cmd *cobra.Command, args []string) {
			serverURL, err := cmd.Flags().GetString("server-url")
			if err != nil {
				log.Fatalln("cannot get server URL:", err)
			}

			tui.Start(serverURL)
		},
	}
	tuiCmd.Flags().String("server-url", "http://localhost:8080", "Server URL for posting ISBNs.")

	rootCmd.AddCommand(tuiCmd)

	rootCmd.AddCommand(serveCmd, waitCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
