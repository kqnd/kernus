package components

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type StatusBar struct {
	View        *tview.TextView
	machineMode bool
	focusPane   string
	hasSelected bool
}

func NewStatusBar(machineMode bool) *StatusBar {
	sb := &StatusBar{
		View:        tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		machineMode: machineMode,
		focusPane:   "",
	}

	sb.View.SetBackgroundColor(ColorZinc900)
	sb.refreshText()
	return sb
}

// SetFocusPane updates the visible focus hint (e.g. Containers, Details).
func (sb *StatusBar) SetFocusPane(pane string) {
	sb.focusPane = pane
	sb.refreshText()
}

// SetHasSelected indicates if a container/machine is selected in the list.
func (sb *StatusBar) SetHasSelected(selected bool) {
	sb.hasSelected = selected
	sb.refreshText()
}

// key returns a formatted shortcut hint: key in gray, label in white.
func key(k, label string) string {
	return fmt.Sprintf("[gray]%s[white] %s", k, label)
}

func (sb *StatusBar) refreshText() {
	sep := "  "

	focus := ""
	if strings.TrimSpace(sb.focusPane) != "" {
		focus = fmt.Sprintf(" [::b]%s[-:-:-]", sb.focusPane)
	}

	var actions string
	switch {
	case sb.focusPane == "Containers" && sb.hasSelected:
		actions = sep + strings.Join([]string{
			key("s", "start"),
			key("t", "stop"),
			key("p", "pause"),
			key("u", "unpause"),
			key("d", "remove"),
		}, sep)
	case sb.focusPane == "Details":
		actions = sep + strings.Join([]string{
			key("1-5", "tabs"),
			key("←/→", "switch tab"),
		}, sep)
	}

	global := sep + strings.Join([]string{
		key("Tab", "switch"),
		key("r", "refresh"),
		key("i", "profile"),
		key("?", "help"),
		key("q", "quit"),
	}, sep)

	sb.View.SetText(focus + actions + global)
}

func (sb *StatusBar) SetError(msg string) {
	sb.View.SetText(" [white::b]Error:[-:-:-] [gray]" + msg + "[white]")
}
