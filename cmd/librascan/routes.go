package main

import (
	"database/sql"

	"github.com/gouthamve/librascan/pkg/handlers"
	"github.com/labstack/echo/v4"
)

// SetupRoutes registers HTTP endpoints using the Echo instance.
func SetupRoutes(e *echo.Echo, database *sql.DB) {

	// Root endpoint that displays all books in an HTML table

	ls := handlers.NewLibrascan(database)

	e.GET("/", ls.GenerateHTMLHandler)
	e.GET("/debug/lookup/:isbn", ls.LookupBookHandler)

	e.POST("/books/:isbn", ls.AddBookFromISBN)
	e.GET("/books/:isbn", ls.GetBookByISBN)
	e.GET("/books", ls.GetAllBooks)
	e.DELETE("/books/:isbn", ls.DeleteBookByISBN)

	e.GET("/shelf/:id", ls.LookupShelfNameHandler)

	e.POST("/books/borrow", ls.BorrowBookByISBN)

	e.GET("/people", ls.GetPeople)
}
