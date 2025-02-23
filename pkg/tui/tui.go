package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Start(startURL string) {
	// Start the TUI
	app := tview.NewApplication()
	flex := tview.NewFlex()
	flex.SetBorder(true).SetTitle("Librascan").SetTitleAlign(tview.AlignCenter)
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			count := flex.GetItemCount()
			for i := 0; i < count; i++ {
				if flex.GetItem(i).HasFocus() {
					app.SetFocus(flex.GetItem((i + 1) % count))
					break
				}
			}
			return nil
		}

		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
			count := flex.GetItemCount()
			if count == 1 {
				return event
			}

			secondFlexItem, ok := flex.GetItem(1).(*tview.Flex)
			if !ok || secondFlexItem.GetItemCount() == 1 {
				flex.RemoveItem(flex.GetItem(1))
				app.SetFocus(flex)
				return nil
			}

			secondFlexItem.RemoveItem(secondFlexItem.GetItem(secondFlexItem.GetItemCount() - 1))
			app.SetFocus(secondFlexItem)
			return nil
		}

		return event
	})

	modeList := tview.NewList()
	modeList.
		AddItem("Manage Books", "Press l to list books", 'l', func() {
			setSecondItem(flex, listBooks(startURL, app))
		}).
		AddItem("Manage Shelves", "Press s to list shelves", 's', nil).
		AddItem("Manage Borrowings", "Press b to list borrowings", 'b', nil).
		AddItem("Quit", "Press q to quit", 'q', func() {
			app.Stop()
		})

	modeList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.Stop()
			return nil
		}

		return event
	})

	modeList.SetBorder(true).SetTitle("Commands").SetTitleAlign(tview.AlignCenter)

	flex.AddItem(
		modeList, 0, 1, true,
	)

	if err := app.SetRoot(flex, true).SetFocus(flex).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func setSecondItem(flex *tview.Flex, item tview.Primitive) {
	if flex.GetItemCount() > 1 {
		flex.RemoveItem(flex.GetItem(1))
	}

	flex.AddItem(item, 0, 4, true)
	// Set the second item
}
