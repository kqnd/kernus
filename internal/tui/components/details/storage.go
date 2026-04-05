package details

import (
	"strings"

	"github.com/kiev/kernus/internal/models"
)

func RenderStorage(b *strings.Builder, c *models.Container) {
	b.WriteString("\n  [white::b]Mounts[white:-:-]\n\n")

	if len(c.Mounts) > 0 {
		tb := NewTableBuilder("Source", "Destination", "Type", "Mode")
		for _, m := range c.Mounts {
			src := m.Source
			if len(src) > 30 {
				src = "..." + src[len(src)-27:]
			}
			dst := m.Destination
			if len(dst) > 30 {
				dst = "..." + dst[len(dst)-27:]
			}
			mode := m.Mode
			if mode == "" {
				mode = "rw"
			}
			tb.AddRow(src, dst, m.Type, mode)
		}
		tb.Render(b)
	} else {
		b.WriteString("  [gray]No mounts[white]\n")
	}

	b.WriteString("\n  [white::b]Block I/O Stats[white:-:-]\n\n")

	if c.Stats != nil {
		bio := c.Stats.BlockIO
		tb := NewTableBuilder("Operation", "Bytes", "Operations")
		tb.AddRow("Read", FormatBytesInt64(bio.ReadBytes), FormatNumber(uint64(bio.ReadOps)))
		tb.AddRow("Write", FormatBytesInt64(bio.WriteBytes), FormatNumber(uint64(bio.WriteOps)))
		tb.Render(b)
	} else {
		b.WriteString("  [gray]Block I/O stats not available[white]\n")
	}
}
