package components

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

type Header struct {
	View           *tview.TextView
	username       string
	role           string
	connected      bool
	sessionWarning string
	app            *tview.Application
	done           chan struct{}
}

func NewHeader(username, role string, app *tview.Application) *Header {
	h := &Header{
		View:     tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		username: username,
		role:     role,
		app:      app,
		done:     make(chan struct{}),
	}

	h.View.SetBackgroundColor(ColorZinc900)
	h.render()
	h.startClock()
	return h
}

func (h *Header) SetConnected(connected bool) {
	h.connected = connected
	h.render()
}

func (h *Header) SetUser(username, role string) {
	h.username = username
	h.role = role
	h.render()
}

func (h *Header) SetSessionWarning(warning string) {
	h.sessionWarning = warning
	h.render()
}

func (h *Header) Stop() {
	select {
	case <-h.done:
	default:
		close(h.done)
	}
}

func (h *Header) render() {
	connIcon := "[gray]○ disconnected[white]"
	if h.connected {
		connIcon = "[white]● connected[white]"
	}

	userInfo := ""
	if h.username != "" {
		userInfo = fmt.Sprintf(" [white]%s", h.username)
		if h.role != "" {
			userInfo += fmt.Sprintf(" [gray][%s][white]", h.role)
		}
	}

	timeStr := time.Now().Format("15:04:05")

	text := fmt.Sprintf(" [white::b]KERNUS[white]%s | %s | %s",
		userInfo, timeStr, connIcon)

	if h.sessionWarning != "" {
		text += fmt.Sprintf(" | [gray]⚠ %s[white]", h.sessionWarning)
	}

	h.View.SetText(text)
}

func (h *Header) startClock() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-h.done:
				return
			case <-ticker.C:
				h.app.QueueUpdateDraw(func() {
					h.render()
				})
			}
		}
	}()
}
