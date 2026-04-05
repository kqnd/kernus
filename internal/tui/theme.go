package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ApplyZincTheme sets global tview colors to a black/zinc/white scheme.
func ApplyZincTheme() {
	zinc950 := tcell.NewRGBColor(9, 9, 11)
	zinc900 := tcell.NewRGBColor(24, 24, 27)
	zinc800 := tcell.NewRGBColor(39, 39, 42)
	zinc600 := tcell.NewRGBColor(82, 82, 91)
	zinc400 := tcell.NewRGBColor(161, 161, 170)

	tview.Styles.PrimitiveBackgroundColor = zinc950
	tview.Styles.ContrastBackgroundColor = zinc900
	tview.Styles.MoreContrastBackgroundColor = zinc800
	tview.Styles.BorderColor = zinc600
	tview.Styles.TitleColor = tcell.ColorWhite
	tview.Styles.GraphicsColor = zinc400
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = zinc400
	tview.Styles.TertiaryTextColor = zinc400
	tview.Styles.InverseTextColor = tcell.ColorWhite
	tview.Styles.ContrastSecondaryTextColor = zinc400
}
