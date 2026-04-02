package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type StatusBar struct {
	View *tview.TextView
}

func NewStatusBar(machineMode bool) *StatusBar {
	sb := &StatusBar{
		View: tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
	}

	sb.View.SetBackgroundColor(tcell.ColorDarkSlateGray)

	if machineMode {
		sb.View.SetText(" [yellow][Tab[][white] Focus  [yellow][r[][white] Refresh  [yellow][o[][white] Onboarding  [yellow][i[][white] Profile  [yellow][?[][white] Help  [yellow][q[][white] Quit")
	} else {
		sb.View.SetText(" [yellow][Tab[][white] Focus  [yellow][s[][white] Start  [yellow][t[][white] Stop  [yellow][p[][white] Pause  [yellow][d[][white] Remove  [yellow][r[][white] Refresh  [yellow][o[][white] Onboarding  [yellow][i[][white] Profile  [yellow][?[][white] Help  [yellow][q[][white] Quit")
	}

	return sb
}

func (sb *StatusBar) SetError(msg string) {
	sb.View.SetText(" [red]Error: " + msg + "[white]")
}
