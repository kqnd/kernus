package details

import (
	"fmt"
	"strings"
)

func BuildBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	color := "green"
	if percent >= 80 {
		color = "red"
	} else if percent >= 50 {
		color = "yellow"
	}

	bar := fmt.Sprintf("[%s]%s[gray]%s[white]",
		color,
		strings.Repeat("█", filled),
		strings.Repeat("░", width-filled),
	)
	return bar
}

func BuildBarWithLabel(label string, percent float64, value string, width int) string {
	return fmt.Sprintf("    %-10s %s %.1f%% %s", label, BuildBar(percent, width), percent, value)
}

func BuildRelativeBars(label1 string, val1 uint64, label2 string, val2 uint64, width int) (string, string) {
	maxVal := val1
	if val2 > maxVal {
		maxVal = val2
	}
	if maxVal == 0 {
		maxVal = 1
	}

	pct1 := float64(val1) / float64(maxVal) * 100
	pct2 := float64(val2) / float64(maxVal) * 100

	line1 := fmt.Sprintf("    %-10s %s %s", label1, BuildBar(pct1, width), FormatBytes(val1))
	line2 := fmt.Sprintf("    %-10s %s %s", label2, BuildBar(pct2, width), FormatBytes(val2))
	return line1, line2
}
