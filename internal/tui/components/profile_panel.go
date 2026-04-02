package components

import (
	"fmt"
	"strings"

	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/models"
	"github.com/rivo/tview"
)

type ProfilePanel struct {
	Modal    *tview.Flex
	OnClose  func()
	OnLogout func()
}

func NewProfilePanel(user *models.User, session *models.Session) *ProfilePanel {
	pp := &ProfilePanel{}

	content := tview.NewTextView().
		SetDynamicColors(true).
		SetText(pp.buildContent(user, session))

	buttons := tview.NewForm()
	buttons.AddButton("Close", func() {
		if pp.OnClose != nil {
			pp.OnClose()
		}
	})
	buttons.AddButton("Logout", func() {
		if pp.OnLogout != nil {
			go func() {
				_ = auth.DeleteSession()
				pp.OnLogout()
			}()
		}
	})
	buttons.SetBorderPadding(0, 0, 0, 0)

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, false).
		AddItem(buttons, 1, 0, true)
	inner.SetBorder(true).SetTitle(" User Profile ").SetTitleAlign(tview.AlignCenter)

	pp.Modal = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(inner, 14, 0, true).
			AddItem(nil, 0, 1, false),
			50, 0, true).
		AddItem(nil, 0, 1, false)

	return pp
}

func (pp *ProfilePanel) buildContent(user *models.User, session *models.Session) string {
	var b strings.Builder

	roleColor := "gray"
	switch user.Role {
	case models.RoleAdmin:
		roleColor = "green"
	case models.RoleOperator:
		roleColor = "yellow"
	}

	fmt.Fprintf(&b, "\n  [yellow]Username[white] : %s\n", user.Username)
	fmt.Fprintf(&b, "  [yellow]Email[white]    : %s\n", user.Email)
	fmt.Fprintf(&b, "  [yellow]Role[white]     : [%s]%s[white]\n", roleColor, user.Role)
	fmt.Fprintf(&b, "  [yellow]Groups[white]   : %s\n", strings.Join(user.Groups, ", "))
	fmt.Fprintf(&b, "\n  [yellow]Current Session:[white]\n")
	if session != nil {
		fmt.Fprintf(&b, "  Expires at : %s\n", session.FormatExpiry())
	}

	return b.String()
}
