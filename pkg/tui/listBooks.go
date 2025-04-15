package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/gouthamve/librascan/pkg/models"
)

func listBooks(serverURL string, app *tview.Application) tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetTitle("Books").SetTitleAlign(tview.AlignCenter)

	go getAndRenderBooks(serverURL, app, flex)

	return flex
}

func getAndRenderBooks(serverURL string, app *tview.Application, flex *tview.Flex) {
	loadingBooks := tview.NewTextView().SetText("Loading Books").SetTextAlign(tview.AlignCenter)
	flex.
		AddItem(nil, 0, 1, false).
		AddItem(loadingBooks, 0, 1, true).
		AddItem(nil, 0, 1, false)

	// Fetch the books
	resp, err := http.Get(serverURL + "/books")
	if err != nil {
		loadingBooks.SetText("Error fetching books: " + err.Error())
		return
	}

	books := []models.Book{}
	if err := json.NewDecoder(resp.Body).Decode(&books); err != nil {
		loadingBooks.SetText("Error decoding books: " + err.Error())
		return
	}

	searchIndex, err := indexBooks(books)
	if err != nil {
		loadingBooks.SetText("Error indexing books: " + err.Error())
		return
	}

	renderBooks(searchIndex, books, flex, app, serverURL)
}

func renderBooks(searchIndex *bookIndex, books []models.Book, flex *tview.Flex, app *tview.Application, serverURL string) {
	table := bookTable(books)
	table.SetSelectedFunc(selectedFunc(books, flex, app, serverURL))

	flex.Clear()

	flex.AddItem(table, 0, 1, true)
	app.Draw()

	app.SetFocus(flex)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '/' {
			if flex.GetItemCount() != 1 || flex.GetItem(0) != table {
				return event
			}

			inputField := tview.NewInputField().SetLabel("Search").SetFieldWidth(20)
			flex.AddItem(inputField, 1, 0, true)
			app.SetFocus(inputField)

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key != tcell.KeyEnter {
					return
				}

				input := inputField.GetText()
				input = strings.TrimPrefix(input, "/")

				query := bleve.NewMatchQuery(input)
				query.Fuzziness = 1
				matchingBooks, err := searchIndex.Search(bleve.NewSearchRequest(query))
				flex.RemoveItem(inputField)
				if err != nil {
					errorText := tview.NewTextView().SetText("Error searching books: " + err.Error()).SetTextAlign(tview.AlignCenter)
					flex.AddItem(errorText, 1, 1, true)
					app.SetFocus(flex)
					return
				}
				if len(matchingBooks) == 0 {
					noResults := tview.NewTextView().SetText("No results found").SetTextAlign(tview.AlignCenter)
					flex.AddItem(noResults, 1, 1, true)
					app.SetFocus(flex)
					return
				}

				app.SetFocus(flex)
				go renderBooks(searchIndex, matchingBooks, flex, app, serverURL)
			})
		}

		return event
	})
}

func bookTable(books []models.Book) *tview.Table {
	// Display the books
	table := tview.NewTable().SetBorders(true).SetSelectable(true, false).SetFixed(1, 0)
	columns := []string{"ISBN", "Title", "Authors", "Published Date", "Categories", "Pages", "Language", "Shelf Name", "Row Number"}

	// Header row.
	for i, col := range columns {
		table.SetCell(0, i, tview.NewTableCell(col).
			SetAlign(tview.AlignCenter),
		)
	}

	for i, book := range books {
		table.SetCell(i+1, 0, tview.NewTableCell(strconv.Itoa(book.ISBN)))
		table.SetCell(i+1, 1, tview.NewTableCell(book.Title))
		table.SetCell(i+1, 2, tview.NewTableCell(strings.Join(book.Authors, ", ")))
		table.SetCell(i+1, 3, tview.NewTableCell(book.PublishedDate))
		table.SetCell(i+1, 4, tview.NewTableCell(strings.Join(book.Categories, ", ")))
		table.SetCell(i+1, 5, tview.NewTableCell(strconv.Itoa(book.Pages)))
		table.SetCell(i+1, 6, tview.NewTableCell(book.Language))
		table.SetCell(i+1, 7, tview.NewTableCell(book.ShelfName))
		table.SetCell(i+1, 8, tview.NewTableCell(strconv.Itoa(book.RowNumber)))
	}

	return table
}

func selectedFunc(books []models.Book, flex *tview.Flex, app *tview.Application, serverURL string) func(row int, _ int) {
	return func(row int, _ int) {
		if row == 0 {
			return
		}

		row--
		if row >= len(books) {
			return
		}

		book := books[row]
		modal := tview.NewModal()
		modal.
			SetText(fmt.Sprintf("Selected book: %s with ISBN: %d", book.Title, book.ISBN)).
			AddButtons([]string{"Delete", "Borrow"}).
			SetDoneFunc(func(_ int, buttonLabel string) {
				if buttonLabel == "Delete" {
					handleDeleteModal(book, modal, flex, app, serverURL)
					return
				}

				if buttonLabel == "Borrow" {
					handleBorrowModal(book, flex, app, serverURL)
					return
				}
			})

		flex.AddItem(modal, 0, 1, true)
		app.SetFocus(modal)
	}
}

func handleDeleteModal(book models.Book, modal *tview.Modal, flex *tview.Flex, app *tview.Application, serverURL string) {
	req, err := http.NewRequest(http.MethodDelete, serverURL+"/books/"+strconv.Itoa(book.ISBN), nil)
	if err != nil {
		modal.SetText("Error deleting book: " + err.Error())
		return
	}

	// Delete the book
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		modal.SetText("Error deleting book: " + err.Error())
		return
	}

	if resp.StatusCode != http.StatusNoContent {
		modal.SetText("Error deleting book: " + resp.Status)
		return
	}

	modal.SetText("Book deleted successfully")
	flex.Clear()
	go getAndRenderBooks(serverURL, app, flex)
}

func handleBorrowModal(book models.Book, flex *tview.Flex, app *tview.Application, serverURL string) {
	flex.Clear()
	loadingPeople := tview.NewTextView().SetText("Loading People").SetTextAlign(tview.AlignCenter)
	flex.AddItem(loadingPeople, 0, 1, true)
	app.SetFocus(flex)

	resp, err := http.Get(serverURL + "/people")
	if err != nil {
		loadingPeople.SetText("Error fetching people: " + err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		loadingPeople.SetText("Error fetching people. Status: " + resp.Status)
		return
	}

	people := []models.Person{}
	if err := json.NewDecoder(resp.Body).Decode(&people); err != nil {
		loadingPeople.SetText("Error decoding people: " + err.Error())
		return
	}

	inputField := tview.NewInputField().SetLabel("Person Name").SetFieldWidth(20)
	inputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
		for _, person := range people {
			if strings.Contains(strings.ToLower(person.Name), strings.ToLower(currentText)) {
				entries = append(entries, person.Name)
			}
		}
		return
	})

	flex.Clear()

	isbn := book.ISBN

	form := tview.NewForm()
	form.SetTitle("Borrow Book").SetTitleAlign(tview.AlignCenter)
	form.AddInputField("ISBN", strconv.Itoa(isbn), 20, nil, nil)
	form.AddFormItem(inputField)

	form.AddButton("Borrow", func() {
		personName := inputField.GetText()
		req := models.BorrowRequest{
			ISBN:       isbn,
			PersonName: personName,
		}
		flex.Clear()
		loadingPeople.SetText("Borrowing book...")
		flex.AddItem(loadingPeople, 0, 1, true)
		app.SetFocus(flex)

		reqBytes, err := json.Marshal(req)
		if err != nil {
			loadingPeople.SetText("Error borrowing book: " + err.Error())
			return
		}

		resp, err := http.Post(serverURL+"/books/borrow", "application/json", strings.NewReader(string(reqBytes)))
		if err != nil {
			loadingPeople.SetText("Error borrowing book: " + err.Error())
			return
		}

		if resp.StatusCode != http.StatusNoContent {
			loadingPeople.SetText("Error borrowing book. Status: " + resp.Status)
			return
		}

		loadingPeople.SetText("Book borrowed successfully")
		flex.Clear()
		go getAndRenderBooks(serverURL, app, flex)
	})
	form.AddButton("Cancel", func() {
		flex.Clear()
		go getAndRenderBooks(serverURL, app, flex)
	})
	flex.AddItem(form, 0, 1, true)
	app.SetFocus(form)
}

func indexBooks(books []models.Book) (*bookIndex, error) {
	booksByISBN := map[string]models.Book{}

	mapping := bleve.NewIndexMapping()

	bleveIndex, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}

	b := bleveIndex.NewBatch()
	for _, book := range books {
		if err := b.Index(strconv.Itoa(book.ISBN), book); err != nil {
			return nil, err
		}

		booksByISBN[strconv.Itoa(book.ISBN)] = book
	}

	if err := bleveIndex.Batch(b); err != nil {
		return nil, err
	}

	return &bookIndex{
		searchIndex: bleveIndex,
		books:       booksByISBN,
	}, nil
}

type bookIndex struct {
	searchIndex bleve.Index
	books       map[string]models.Book
}

func (b *bookIndex) Search(req *bleve.SearchRequest) ([]models.Book, error) {
	searchResults, err := b.searchIndex.Search(req)
	if err != nil {
		return nil, err
	}

	var books []models.Book
	for _, hit := range searchResults.Hits {
		book, ok := b.books[hit.ID]
		if !ok {
			continue
		}

		books = append(books, book)
	}

	return books, nil
}
