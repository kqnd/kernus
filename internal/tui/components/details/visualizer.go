package details

import (
	"fmt"
	"strings"

	"github.com/kqnd/kernus/internal/models"
)

type StatsVisualizer struct {
	formatter *Formatter
}

func NewStatsVisualizer() *StatsVisualizer {
	return &StatsVisualizer{
		formatter: NewFormatter(),
	}
}

func (v *StatsVisualizer) BuildProgressBar(percentage float64, width int, label string) string {
	if width < 10 {
		width = 20
	}

	filled := int((percentage / 100.0) * float64(width))
	if filled > width {
		filled = width
	}

	color := v.formatter.GetUsageColor(percentage)
	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)

	return fmt.Sprintf("[%s]%s[gray]%s[white] %.1f%% %s",
		color, bar, empty, percentage, label)
}

func (v *StatsVisualizer) BuildMemoryVisualization(mem models.ContainerMemory) string {
	if mem.Limit == 0 {
		return "[gray]No memory limit set[white]"
	}

	usagePercentage := mem.Percentage()
	cachePercentage := float64(mem.Cache) / float64(mem.Limit) * 100
	rssPercentage := float64(mem.RSS) / float64(mem.Limit) * 100

	var result strings.Builder
	result.WriteString("  Memory Layout:\n")
	result.WriteString(fmt.Sprintf("  %s\n", v.BuildProgressBar(usagePercentage, 40, "Total")))
	result.WriteString(fmt.Sprintf("  %s\n", v.BuildProgressBar(rssPercentage, 40, "RSS")))
	result.WriteString(fmt.Sprintf("  %s", v.BuildProgressBar(cachePercentage, 40, "Cache")))

	return result.String()
}

func (v *StatsVisualizer) BuildCPUVisualization(cpu models.ContainerCPU) string {
	var result strings.Builder

	result.WriteString("  CPU Usage:\n")
	result.WriteString(fmt.Sprintf("  %s\n", v.BuildProgressBar(cpu.Usage, 40, fmt.Sprintf("(%d cores)", cpu.Cores))))

	if cpu.Throttling.Periods > 0 {
		throttlePercentage := float64(cpu.Throttling.ThrottledPeriods) / float64(cpu.Throttling.Periods) * 100
		result.WriteString(fmt.Sprintf("  %s", v.BuildProgressBar(throttlePercentage, 40, "Throttled")))
	}

	return result.String()
}

func (v *StatsVisualizer) BuildNetworkVisualization(network models.ContainerNetwork) string {
	if network.RxBytes == 0 && network.TxBytes == 0 {
		return "  [gray]No network activity[white]"
	}

	maxBytes := network.RxBytes
	if network.TxBytes > maxBytes {
		maxBytes = network.TxBytes
	}

	rxPercentage := float64(network.RxBytes) / float64(maxBytes) * 100
	txPercentage := float64(network.TxBytes) / float64(maxBytes) * 100

	var result strings.Builder
	result.WriteString("  Network I/O:\n")
	result.WriteString(fmt.Sprintf("  %s\n", v.BuildProgressBar(rxPercentage, 35, fmt.Sprintf("RX %s", v.formatter.FormatBytes(network.RxBytes)))))
	result.WriteString(fmt.Sprintf("  %s", v.BuildProgressBar(txPercentage, 35, fmt.Sprintf("TX %s", v.formatter.FormatBytes(network.TxBytes)))))

	return result.String()
}

func (v *StatsVisualizer) BuildBlockIOVisualization(blockIO models.ContainerBlockIO) string {
	if blockIO.ReadBytes == 0 && blockIO.WriteBytes == 0 {
		return "  [gray]No block I/O activity[white]"
	}

	maxBytes := blockIO.ReadBytes
	if blockIO.WriteBytes > maxBytes {
		maxBytes = blockIO.WriteBytes
	}

	readPercentage := float64(blockIO.ReadBytes) / float64(maxBytes) * 100
	writePercentage := float64(blockIO.WriteBytes) / float64(maxBytes) * 100

	var result strings.Builder
	result.WriteString("  Block I/O:\n")
	result.WriteString(fmt.Sprintf("  %s\n", v.BuildProgressBar(readPercentage, 35, fmt.Sprintf("Read %s", v.formatter.FormatBytes(blockIO.ReadBytes)))))
	result.WriteString(fmt.Sprintf("  %s", v.BuildProgressBar(writePercentage, 35, fmt.Sprintf("Write %s", v.formatter.FormatBytes(blockIO.WriteBytes)))))

	return result.String()
}

func (v *StatsVisualizer) BuildStatsPlaceholder() string {
	return `
[gray]┌─────────────────────────────────┐[white]
[gray]│[white]  CPU Usage    : N/A             [gray]│[white]
[gray]│[white]  Memory Usage : N/A             [gray]│[white]
[gray]│[white]  Network I/O  : N/A             [gray]│[white]
[gray]│[white]  Block I/O    : N/A             [gray]│[white]
[gray]│[white]  PIDs         : N/A             [gray]│[white]
[gray]└─────────────────────────────────┘[white]`
}
