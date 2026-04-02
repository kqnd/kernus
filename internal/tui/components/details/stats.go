package details

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kiev/kernus/internal/models"
)

var (
	statsCache     *models.ContainerStats
	statsCacheID   string
	statsCacheTime time.Time
	statsMu        sync.Mutex
)

func RenderStats(b *strings.Builder, c *models.Container) {
	if c.Stats == nil {
		b.WriteString("\n  [gray]Stats not available for this container[white]\n")
		return
	}

	statsMu.Lock()
	if statsCacheID == c.ID && time.Since(statsCacheTime) < 2*time.Second && statsCache != nil {
		c.Stats = statsCache
	} else {
		statsCache = c.Stats
		statsCacheID = c.ID
		statsCacheTime = time.Now()
	}
	statsMu.Unlock()

	s := c.Stats

	b.WriteString("\n  [yellow::b]CPU Usage[white:-:-]\n")
	b.WriteString(BuildBarWithLabel("Total", s.CPU.Usage, fmt.Sprintf("(%d cores)", s.CPU.Cores), 40))
	b.WriteString("\n")
	if s.CPU.Throttling > 0 {
		b.WriteString(BuildBarWithLabel("Throttle", s.CPU.Throttling, "", 40))
		b.WriteString("\n")
	}

	b.WriteString("\n  [yellow::b]Memory[white:-:-]\n")
	memPct := s.Memory.Percentage()
	b.WriteString(BuildBarWithLabel("Total", memPct, fmt.Sprintf("%s / %s", FormatBytesInt64(s.Memory.Usage), FormatBytesInt64(s.Memory.Limit)), 40))
	b.WriteString("\n")

	if s.Memory.Limit > 0 && s.Memory.RSS > 0 {
		rssPct := float64(s.Memory.RSS) / float64(s.Memory.Limit) * 100
		b.WriteString(BuildBarWithLabel("RSS", rssPct, FormatBytesInt64(s.Memory.RSS), 40))
		b.WriteString("\n")
	}

	if s.Memory.Limit > 0 && s.Memory.Cache > 0 {
		cachePct := float64(s.Memory.Cache) / float64(s.Memory.Limit) * 100
		b.WriteString(BuildBarWithLabel("Cache", cachePct, FormatBytesInt64(s.Memory.Cache), 40))
		b.WriteString("\n")
	}

	b.WriteString("\n  [yellow::b]Network I/O[white:-:-]\n")
	rxLine, txLine := BuildRelativeBars("RX", uint64(s.Network.RxBytes), "TX", uint64(s.Network.TxBytes), 40)
	b.WriteString(rxLine)
	b.WriteString("\n")
	b.WriteString(txLine)
	b.WriteString("\n")

	b.WriteString("\n  [yellow::b]Block I/O[white:-:-]\n")
	readLine, writeLine := BuildRelativeBars("Read", uint64(s.BlockIO.ReadBytes), "Write", uint64(s.BlockIO.WriteBytes), 40)
	b.WriteString(readLine)
	b.WriteString("\n")
	b.WriteString(writeLine)
	b.WriteString("\n")

	b.WriteString("\n  [yellow::b]Process Info[white:-:-]\n")
	fmt.Fprintf(b, "    PIDs     : %d\n", s.PIDs)
}
