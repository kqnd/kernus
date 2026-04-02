package components

import (
	"context"
	"fmt"

	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/models"
	"github.com/rivo/tview"
)

type LoginForm struct {
	Form      *tview.Form
	ErrorText *tview.TextView
	OnSuccess func(session *models.Session)
	OnCancel  func()
}

func NewLoginForm() *LoginForm {
	lf := &LoginForm{}

	lf.ErrorText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	lf.Form = tview.NewForm()
	lf.Form.SetBorder(true).SetTitle(" Login ").SetTitleAlign(tview.AlignCenter)
	lf.Form.AddInputField("Username", "", 40, nil, nil)
	lf.Form.AddPasswordField("Password", "", 40, '*', nil)
	lf.Form.AddButton("Login", func() {
		go lf.doLogin()
	})
	lf.Form.AddButton("Quit", func() {
		if lf.OnCancel != nil {
			lf.OnCancel()
		}
	})

	return lf
}

func (lf *LoginForm) doLogin() {
	usernameItem := lf.Form.GetFormItemByLabel("Username")
	passwordItem := lf.Form.GetFormItemByLabel("Password")
	if usernameItem == nil || passwordItem == nil {
		return
	}
	username := usernameItem.(*tview.InputField).GetText()
	password := passwordItem.(*tview.InputField).GetText()

	if username == "" || password == "" {
		lf.ErrorText.SetText("[red]✗ Username and password are required[white]")
		return
	}

	client := auth.NewDefaultClient()
	session, err := client.Login(context.TODO(), username, password)
	if err != nil {
		lf.ErrorText.SetText(fmt.Sprintf("[red]✗ %v[white]", err))
		return
	}

	saveErr := auth.SaveSession(&auth.StoredSession{
		Token:     session.Token,
		Username:  session.Username,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
	})
	if saveErr != nil {
		lf.ErrorText.SetText(fmt.Sprintf("[red]✗ %v[white]", saveErr))
		return
	}

	if lf.OnSuccess != nil {
		lf.OnSuccess(session)
	}
}
