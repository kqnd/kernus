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
	content  *tview.TextView
	session  *models.Session
	OnClose  func()
	OnLogout func()
}

func NewProfilePanel(user *models.User, session *models.Session) *ProfilePanel {
	pp := &ProfilePanel{session: session}

	pp.content = tview.NewTextView().
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
		AddItem(pp.content, 0, 1, false).
		AddItem(buttons, 1, 0, true)
	inner.SetBorder(true).SetTitle(" Profile ").SetTitleAlign(tview.AlignCenter)

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

// SetUser refreshes the profile body (e.g. after API load).
func (pp *ProfilePanel) SetUser(user *models.User) {
	if pp.content == nil || user == nil {
		return
	}
	pp.content.SetText(pp.buildContent(user, pp.session))
}

func (pp *ProfilePanel) buildContent(user *models.User, session *models.Session) string {
	var b strings.Builder

	roleStr := string(user.Role)
	if roleStr == "" {
		roleStr = "—"
	}

	groups := "—"
	if len(user.Groups) > 0 {
		groups = strings.Join(user.Groups, ", ")
	}

	fmt.Fprintf(&b, "\n  [gray]Username[white] : %s\n", user.Username)
	fmt.Fprintf(&b, "  [gray]Email[white]    : %s\n", user.Email)
	fmt.Fprintf(&b, "  [gray]Role[white]     : %s\n", roleStr)
	if len(user.Groups) > 0 {
		fmt.Fprintf(&b, "  [gray]Groups[white]   : %s\n", groups)
	}
	fmt.Fprintf(&b, "\n  [white::b]Session[-:-:-]\n")
	if session != nil {
		fmt.Fprintf(&b, "  [gray]Expires[white]  : %s\n", session.FormatExpiry())
	}

	return b.String()
}
