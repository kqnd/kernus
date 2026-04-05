package details

import (
	"fmt"
	"strings"
)

type TableBuilder struct {
	headers []string
	rows    [][]string
	widths  []int
}

func NewTableBuilder(headers ...string) *TableBuilder {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &TableBuilder{
		headers: headers,
		widths:  widths,
	}
}

func (t *TableBuilder) AddRow(values ...string) {
	row := make([]string, len(t.headers))
	for i := 0; i < len(t.headers) && i < len(values); i++ {
		row[i] = values[i]
		if len(values[i]) > t.widths[i] {
			t.widths[i] = len(values[i])
		}
	}
	t.rows = append(t.rows, row)
}

func (t *TableBuilder) Render(b *strings.Builder) {
	if len(t.rows) == 0 {
		b.WriteString("  [gray]No data available[white]\n")
		return
	}

	topLine := t.buildLine("┌", "┬", "┐")
	midLine := t.buildLine("├", "┼", "┤")
	botLine := t.buildLine("└", "┴", "┘")

	b.WriteString("  ")
	b.WriteString(topLine)
	b.WriteString("\n")

	b.WriteString("  │")
	for i, h := range t.headers {
		fmt.Fprintf(b, " %-*s │", t.widths[i], h)
	}
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(midLine)
	b.WriteString("\n")

	for _, row := range t.rows {
		b.WriteString("  │")
		for i, cell := range row {
			fmt.Fprintf(b, " %-*s │", t.widths[i], cell)
		}
		b.WriteString("\n")
	}

	b.WriteString("  ")
	b.WriteString(botLine)
	b.WriteString("\n")
}

func (t *TableBuilder) buildLine(left string, mid string, right string) string {
	parts := make([]string, len(t.widths))
	for i, w := range t.widths {
		parts[i] = strings.Repeat("─", w+2)
	}
	return left + strings.Join(parts, mid) + right
}
