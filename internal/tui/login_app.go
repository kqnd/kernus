package tui

import (
	"context"
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/models"
	"github.com/rivo/tview"
)

func RunLoginApp() error {
	ApplyZincTheme()

	app := tview.NewApplication()
	var session *models.Session
	var loginErr error
	var mu sync.Mutex

	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("\n[white::b]K E R N U S[white]\n[gray]Infrastructure Monitor")

	errorText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Login ").SetTitleAlign(tview.AlignCenter)

	form.AddInputField("Username", "", 40, nil, nil)
	form.AddPasswordField("Password", "", 40, '*', nil)

	doLogin := func() {
		usernameItem := form.GetFormItemByLabel("Username")
		passwordItem := form.GetFormItemByLabel("Password")
		if usernameItem == nil || passwordItem == nil {
			return
		}
		username := usernameItem.(*tview.InputField).GetText()
		password := passwordItem.(*tview.InputField).GetText()

		if username == "" || password == "" {
			app.QueueUpdateDraw(func() {
				errorText.SetText("[gray]Username and password are required[white]")
			})
			return
		}

		client := auth.NewDefaultClient()
		s, err := client.Login(context.TODO(), username, password)
		if err != nil {
			app.QueueUpdateDraw(func() {
				errorText.SetText(fmt.Sprintf("[gray]%v[white]", err))
			})
			return
		}

		err = auth.SaveSession(&auth.StoredSession{
			Token:     s.Token,
			Username:  s.Username,
			Email:     s.Username,
			UserID:    s.UserID,
			ExpiresAt: s.ExpiresAt,
		})
		if err != nil {
			mu.Lock()
			loginErr = err
			mu.Unlock()
			app.Stop()
			return
		}

		mu.Lock()
		session = s
		mu.Unlock()
		app.Stop()
	}

	form.AddButton("Login", func() {
		go doLogin()
	})
	form.AddButton("Quit", func() {
		app.Stop()
	})

	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[gray][Tab] Navigate  [Enter] Confirm  [Esc] Quit[white]")

	formLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(title, 4, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 0, true).
			AddItem(nil, 0, 1, false),
			10, 0, true).
		AddItem(errorText, 2, 0, false).
		AddItem(helpText, 1, 0, false).
		AddItem(nil, 0, 1, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		return event
	})

	err := app.SetRoot(formLayout, true).EnableMouse(false).Run()
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	if loginErr != nil {
		return loginErr
	}

	if session == nil {
		return fmt.Errorf("login cancelled")
	}

	return nil
}
