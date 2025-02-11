package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
)

func readBookInfo(serverURL string) {
	shelf, rowNumber, err := getShelfFromCode(serverURL, "00000")
	if err != nil {
		log.Fatalln("Cannot get shelf:", err)
	}

	for {
		fmt.Print("Enter ISBN13 or shelfCode: ")
		var isbn string
		fmt.Scanln(&isbn)
		if len(isbn) != 13 && len(isbn) != 5 {
			fmt.Println("Invalid ISBN")
			continue
		}

		if len(isbn) == 5 {
			prevShelf := shelf
			prevRow := rowNumber

			shelf, rowNumber, err = getShelfFromCode(serverURL, isbn)
			if err != nil {
				slog.Error("cannot get shelf; using previous shelf", "error", err, "prev_shelf", prevShelf.Name)
				shelf = prevShelf
				rowNumber = prevRow
			} else {
				slog.Info("Shelf changed", "shelf", shelf.Name, "row", rowNumber)
			}

			continue
		}

		if len(isbn) != 13 {
			fmt.Println("Invalid ISBN")
			continue
		}

		fmt.Println("ISBN:", isbn, "Shelf:", shelf.Name, "Row:", rowNumber)
		ingestBook(serverURL, isbn, shelf.ID, rowNumber)
	}
}

func ingestBook(serverURL, isbn string, shelfID, rowNumber int) {
	// Use the provided serverURL instead of the hardcoded value.
	fullURL := fmt.Sprintf("%s/books/%s?shelf_id=%d&row_number=%d", serverURL, isbn, shelfID, rowNumber)
	resp, err := http.Post(fullURL, "application/json", io.Reader(nil))
	if err != nil {
		slog.Error("cannot post ISBN", "error", err)
	}

	if resp.StatusCode/100 != 2 {
		slog.Error("unexpected status code", "status", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("cannot read response body", "error", err)
		}
		fmt.Println("Body:", string(body))
	} else {
		book := Book{}
		if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
			slog.Error("cannot decode response body", "error", err)
		}

		fmt.Println("Book:", book)
	}

	if err := resp.Body.Close(); err != nil {
		slog.Error("cannot close response body", "error", err)
	}
}

func getShelfFromCode(serverURL, shelfCode string) (Shelf, int, error) {
	// Last char is the row_number.
	rowNumberStr := shelfCode[len(shelfCode)-1:]
	rowNumber, err := strconv.Atoi(rowNumberStr)
	if err != nil {
		return Shelf{}, 0, fmt.Errorf("invalid row number: %w", err)
	}

	shelfID := shelfCode[:len(shelfCode)-1]

	fullURL := fmt.Sprintf("%s/shelf/%s", serverURL, shelfID)
	resp, err := http.Get(fullURL)
	if err != nil {
		return Shelf{}, 0, fmt.Errorf("cannot get shelf: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return Shelf{}, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	shelf := Shelf{}
	json.NewDecoder(resp.Body).Decode(&shelf)
	return shelf, rowNumber, nil
}
