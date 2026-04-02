package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Modal struct {
	visible bool
}

func NewModal() *Modal {
	return &Modal{}
}

func (m *Modal) IsVisible() bool {
	return m.visible
}

func (m *Modal) Show(app *tview.Application, returnTo tview.Primitive, message string, onConfirm func(), onCancel func()) {
	m.visible = true

	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Confirm", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			m.visible = false
			app.SetRoot(returnTo, true)
			if buttonLabel == "Confirm" {
				if onConfirm != nil {
					onConfirm()
				}
			} else {
				if onCancel != nil {
					onCancel()
				}
			}
		})

	modal.SetBorder(true).SetTitle(" Confirm Action ")
	modal.SetBackgroundColor(tcell.ColorDarkSlateGray)

	pages := tview.NewPages().
		AddPage("main", returnTo, true, true).
		AddPage("modal", modal, true, true)

	app.SetRoot(pages, true)
}
