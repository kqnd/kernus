package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kern/internal/models"
	"github.com/rivo/tview"
)

type Details struct {
	view             *tview.TextView
	currentContainer *models.Container
	tabs             []string
	currentTab       int
}

const (
	TAB_OVERVIEW = iota
	TAB_STATS
	TAB_NETWORK
	TAB_STORAGE
)

func NewDetails() *Details {
	d := &Details{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetWordWrap(true).
			SetScrollable(true).
			SetChangedFunc(func() {
			}),
		tabs:       []string{"Overview", "Stats", "Network", "Storage"},
		currentTab: TAB_OVERVIEW,
	}

	d.view.SetBorder(true).SetTitle(" Container Details ")
	d.view.SetText("[yellow]Select a container to view details[white]\n\n[gray]Use arrow keys to navigate between tabs[white]")

	d.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			d.PrevTab()
			return nil
		case tcell.KeyRight:
			d.NextTab()
			return nil
		}
		return event
	})

	return d
}

func (d *Details) ShowContainer(container *models.Container) {
	d.currentContainer = container
	d.updateView()
}

func (d *Details) SwitchTab(tab int) {
	if tab >= 0 && tab < len(d.tabs) {
		d.currentTab = tab
		d.updateView()
	}
}

func (d *Details) NextTab() {
	d.currentTab = (d.currentTab + 1) % len(d.tabs)
	d.updateView()
}

func (d *Details) PrevTab() {
	d.currentTab = (d.currentTab - 1 + len(d.tabs)) % len(d.tabs)
	d.updateView()
}

func (d *Details) updateView() {
	if d.currentContainer == nil {
		d.view.SetTitle(" Container Details ")
		d.view.SetText(d.buildEmptyState())
		return
	}

	title := fmt.Sprintf(" %s - %s ", d.tabs[d.currentTab], d.currentContainer.ShortName())
	d.view.SetTitle(title)

	var content string
	switch d.currentTab {
	case TAB_OVERVIEW:
		content = d.buildOverviewContent()
	case TAB_STATS:
		content = d.buildEnhancedStatsContent()
	case TAB_NETWORK:
		content = d.buildNetworkContent()
	case TAB_STORAGE:
		content = d.buildStorageContent()
	// case TAB_LOGS:
	// 	content = d.buildLogsContent()
	default:
		content = d.buildOverviewContent()
	}

	d.view.SetText(content)
}

func (d *Details) buildEmptyState() string {
	return `[yellow]Container Details[white]

[gray]┌─────────────────────────────────────┐[white]
[gray]│[white]  No container selected              [gray]│[white]
[gray]│[white]                                     [gray]│[white]
[gray]│[white]  Select a container from the list   [gray]│[white]
[gray]│[white]  to view detailed information       [gray]│[white]
[gray]│[white]                                     [gray]│[white]
[gray]│[white]  Available tabs:                    [gray]│[white]
[gray]│[white]  • Overview - Basic info & status   [gray]│[white]
[gray]│[white]  • Stats    - Resource usage        [gray]│[white]
[gray]│[white]  • Network  - Network configuration [gray]│[white]
[gray]│[white]  • Storage  - Mounts & volumes      [gray]│[white]
[gray]│[white]  • Logs     - Container logs        [gray]│[white]
[gray]└─────────────────────────────────────┘[white]

[darkgray]Use [white]Left/Right arrows[darkgray] to switch between tabs[white]`
}

func (d *Details) buildTabHeader() string {
	var tabs []string
	for i, tab := range d.tabs {
		if i == d.currentTab {
			tabs = append(tabs, fmt.Sprintf("[white]> %s <[white]", tab))
		} else {
			tabs = append(tabs, fmt.Sprintf("[gray]  %s  [white]", tab))
		}
	}
	return fmt.Sprintf("%s\n\n", strings.Join(tabs, ""))
}

func (d *Details) buildOverviewContent() string {
	c := d.currentContainer

	healthStatus := "Unknown"
	healthColor := "gray"
	healthIcon := ""

	if c.Health != nil {
		healthStatus = string(c.Health.Status)
		healthColor = c.Health.Status.Color()
		healthIcon = c.Health.Status.Icon()
	}

	restartPolicy := "No restart"
	if c.RestartPolicy.Name != "" {
		restartPolicy = c.RestartPolicy.Name
		if c.RestartPolicy.MaximumRetryCount > 0 {
			restartPolicy = fmt.Sprintf("%s (max: %d)", restartPolicy, c.RestartPolicy.MaximumRetryCount)
		}
	}

	return fmt.Sprintf(`%s[yellow]Container Information[white]

[yellow]Identity[white]
  ID       : %s
  Name     : %s
  Image    : [cyan]%s[white]
  Tag      : [blue]%s[white]

[yellow]Status[white]
  Status   : [%s]%s %s[white]
  State    : %s
  Health   : [%s]%s %s[white]
  Exit Code: %s

[yellow]Timing[white]
  Created  : %s
  Started  : %s
  Age      : %s
  Uptime   : %s

[yellow]Configuration[white]
  Command  : [cyan]%s[white]
  Restart  : %s
  PIDs     : %s

[yellow]Quick Stats[white]
%s

[yellow]Labels (%d)[white]
%s`,
		d.buildTabHeader(),
		c.ShortID(),
		c.ShortName(),
		c.ImageName(),
		c.ImageTag(),
		c.Status.Color(), c.Status.Icon(), c.Status,
		c.State,
		healthColor, healthIcon, healthStatus,
		d.formatExitCode(c.ExitCode),
		d.formatTime(c.Created),
		d.formatTime(c.Started),
		c.FormatAge(),
		c.FormatUptime(),
		d.truncateString(c.Command, 60),
		restartPolicy,
		d.formatPIDs(c),
		d.buildQuickStats(c),
		len(c.Labels),
		d.buildLabelsGrid(c.Labels))
}

func (d *Details) buildEnhancedStatsContent() string {
	c := d.currentContainer

	if c.Stats == nil {
		return fmt.Sprintf(`%s[yellow]Resource Statistics[white]

[red]No Statistics Available[white]

Statistics are only available for running containers.
%s`,
			d.buildTabHeader(),
			d.buildStatsPlaceholder())
	}

	return fmt.Sprintf(`%s[yellow]Resource Statistics[white]

[yellow]CPU Performance[white]
%s

[yellow]Memory Usage[white]
%s

[yellow]Network Activity[white]
%s

[yellow]Storage I/O[white]
%s

[yellow]Process Information[white]
  Active PIDs: [cyan]%d[white] processes

[gray]Last Updated: %s[white]`,
		d.buildTabHeader(),
		d.buildCPUVisualization(c.Stats.CPU),
		d.buildMemoryVisualization(c.Stats.Memory),
		d.buildNetworkVisualization(c.Stats.Network),
		d.buildBlockIOVisualization(c.Stats.BlockIO),
		c.Stats.PIDs,
		d.formatTime(c.Stats.Timestamp))
}

func (d *Details) buildNetworkContent() string {
	c := d.currentContainer

	return fmt.Sprintf(`%s[yellow]Network Configuration[white]

[yellow]Port Mappings (%d)[white]
%s

[yellow]Networks (%d)[white]
%s

[yellow]Network Statistics[white]
%s`,
		d.buildTabHeader(),
		len(c.Ports),
		d.buildPortsTable(c.Ports),
		len(c.Networks),
		d.buildNetworksTable(c.Networks),
		d.buildNetworkStats(c))
}

func (d *Details) buildStorageContent() string {
	c := d.currentContainer

	return fmt.Sprintf(`%s[yellow]Storage Configuration[white]

[yellow]Mounts (%d)[white]
%s

[yellow]Block I/O Statistics[white]
%s`,
		d.buildTabHeader(),
		len(c.Mounts),
		d.buildMountsTable(c.Mounts),
		d.buildBlockIOStats(c))
}

func (d *Details) buildLogsContent() string {
	return fmt.Sprintf(`%s[yellow]Container Logs[white]

[gray]Log viewing functionality would be implemented here.
This would show real-time container logs with:

• Timestamps
• Log levels (if parseable)  
• Search functionality
• Auto-refresh capability
• Color coding by log level

Use docker logs command for now.[white]`,
		d.buildTabHeader())
}

func (d *Details) buildQuickStats(c *models.Container) string {
	if c.Stats == nil {
		return `  CPU      : [gray]N/A (not running)[white]
  Memory   : [gray]N/A (not running)[white]
  Network  : [gray]N/A (not running)[white]
  PIDs     : [gray]N/A (not running)[white]`
	}

	cpuColor := d.getUsageColor(c.Stats.CPU.Usage)
	memColor := d.getUsageColor(c.Stats.Memory.Percentage())

	return fmt.Sprintf(`  CPU      : [%s]%.1f%%[white]
  Memory   : [%s]%.1fMB (%.1f%%)[white]
  Network  : ↓ %s ↑ %s
  PIDs     : %d processes`,
		cpuColor, c.Stats.CPU.Usage,
		memColor, float64(c.Stats.Memory.Usage)/1024/1024, c.Stats.Memory.Percentage(),
		d.formatBytes(c.Stats.Network.RxBytes),
		d.formatBytes(c.Stats.Network.TxBytes),
		c.Stats.PIDs)
}

func (d *Details) buildLabelsGrid(labels map[string]string) string {
	if len(labels) == 0 {
		return "  [gray]No labels defined[white]"
	}

	var result strings.Builder
	count := 0
	maxLabels := 8

	var keys []string
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if count >= maxLabels {
			result.WriteString("  [gray]... and more[white]\n")
			break
		}

		value := labels[key]
		truncatedKey := d.truncateString(key, 20)
		truncatedValue := d.truncateString(value, 30)

		result.WriteString(fmt.Sprintf("  [yellow]%s[white]: %s\n", truncatedKey, truncatedValue))
		count++
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildPortsTable(ports []models.Port) string {
	if len(ports) == 0 {
		return "  [gray]No ports exposed[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Private    Public     Type    IP[white]\n")
	result.WriteString("  [gray]─────────────────────────────────────[white]\n")

	for _, port := range ports {
		publicStr := "─"
		ipStr := "─"

		if port.PublicPort > 0 {
			publicStr = fmt.Sprintf("%d", port.PublicPort)
		}
		if port.IP != "" {
			ipStr = port.IP
		}

		result.WriteString(fmt.Sprintf("  %-9d  %-9s  %-6s  %s\n",
			port.PrivatePort, publicStr, port.Type, ipStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildNetworksTable(networks []models.Network) string {
	if len(networks) == 0 {
		return "  [gray]No networks configured[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Network          IP Address       Gateway[white]\n")
	result.WriteString("  [gray]───────────────────────────────────────────────[white]\n")

	for _, network := range networks {
		ipStr := network.IPAddress
		gatewayStr := network.Gateway

		if ipStr == "" {
			ipStr = "─"
		}
		if gatewayStr == "" {
			gatewayStr = "─"
		}

		result.WriteString(fmt.Sprintf("  %-15s  %-15s  %s\n",
			d.truncateString(network.Name, 15), ipStr, gatewayStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildMountsTable(mounts []models.Mount) string {
	if len(mounts) == 0 {
		return "  [gray]No mounts configured[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Source                    Destination              Type    Mode[white]\n")
	result.WriteString("  [gray]─────────────────────────────────────────────────────────────────────[white]\n")

	for _, mount := range mounts {
		modeStr := mount.Mode
		if modeStr == "" {
			if mount.RW {
				modeStr = "rw"
			} else {
				modeStr = "ro"
			}
		}

		result.WriteString(fmt.Sprintf("  %-24s  %-23s  %-6s  %s\n",
			d.truncateString(mount.Source, 24),
			d.truncateString(mount.Destination, 23),
			mount.Type,
			modeStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildNetworkStats(c *models.Container) string {
	if c.Stats == nil {
		return "  [gray]Network statistics not available[white]"
	}

	return fmt.Sprintf(`  Received : %s (%s packets, %d errors)
  Sent     : %s (%s packets, %d errors)`,
		d.formatBytes(c.Stats.Network.RxBytes),
		d.formatNumber(c.Stats.Network.RxPackets),
		c.Stats.Network.RxErrors,
		d.formatBytes(c.Stats.Network.TxBytes),
		d.formatNumber(c.Stats.Network.TxPackets),
		c.Stats.Network.TxErrors)
}

func (d *Details) buildBlockIOStats(c *models.Container) string {
	if c.Stats == nil {
		return "  [gray]Block I/O statistics not available[white]"
	}

	return fmt.Sprintf(`  Read     : %s (%s operations)
  Write    : %s (%s operations)`,
		d.formatBytes(c.Stats.BlockIO.ReadBytes),
		d.formatNumber(c.Stats.BlockIO.ReadOps),
		d.formatBytes(c.Stats.BlockIO.WriteBytes),
		d.formatNumber(c.Stats.BlockIO.WriteOps))
}

func (d *Details) buildStatsPlaceholder() string {
	return `
[gray]┌─────────────────────────────────┐[white]
[gray]│[white]  CPU Usage    : N/A             [gray]│[white]
[gray]│[white]  Memory Usage : N/A             [gray]│[white]
[gray]│[white]  Network I/O  : N/A             [gray]│[white]
[gray]│[white]  Block I/O    : N/A             [gray]│[white]
[gray]│[white]  PIDs         : N/A             [gray]│[white]
[gray]└─────────────────────────────────┘[white]`
}

func (d *Details) buildProgressBar(percentage float64, width int, label string) string {
	if width < 10 {
		width = 20
	}

	filled := int((percentage / 100.0) * float64(width))
	if filled > width {
		filled = width
	}

	color := d.getUsageColor(percentage)

	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)

	return fmt.Sprintf("[%s]%s[gray]%s[white] %.1f%% %s",
		color, bar, empty, percentage, label)
}

func (d *Details) buildMemoryVisualization(mem models.ContainerMemory) string {
	if mem.Limit == 0 {
		return "[gray]No memory limit set[white]"
	}

	usagePercentage := mem.Percentage()
	cachePercentage := float64(mem.Cache) / float64(mem.Limit) * 100
	rssPercentage := float64(mem.RSS) / float64(mem.Limit) * 100

	var result strings.Builder
	result.WriteString("  Memory Layout:\n")
	result.WriteString(fmt.Sprintf("  %s\n", d.buildProgressBar(usagePercentage, 40, "Total")))
	result.WriteString(fmt.Sprintf("  %s\n", d.buildProgressBar(rssPercentage, 40, "RSS")))
	result.WriteString(fmt.Sprintf("  %s", d.buildProgressBar(cachePercentage, 40, "Cache")))

	return result.String()
}

func (d *Details) buildCPUVisualization(cpu models.ContainerCPU) string {
	var result strings.Builder

	result.WriteString("  CPU Usage:\n")
	result.WriteString(fmt.Sprintf("  %s\n", d.buildProgressBar(cpu.Usage, 40, fmt.Sprintf("(%d cores)", cpu.Cores))))

	if cpu.Throttling.Periods > 0 {
		throttlePercentage := float64(cpu.Throttling.ThrottledPeriods) / float64(cpu.Throttling.Periods) * 100
		result.WriteString(fmt.Sprintf("  %s", d.buildProgressBar(throttlePercentage, 40, "Throttled")))
	}

	return result.String()
}

func (d *Details) buildNetworkVisualization(network models.ContainerNetwork) string {
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
	result.WriteString(fmt.Sprintf("  %s\n", d.buildProgressBar(rxPercentage, 35, fmt.Sprintf("RX %s", d.formatBytes(network.RxBytes)))))
	result.WriteString(fmt.Sprintf("  %s", d.buildProgressBar(txPercentage, 35, fmt.Sprintf("TX %s", d.formatBytes(network.TxBytes)))))

	return result.String()
}

func (d *Details) buildBlockIOVisualization(blockIO models.ContainerBlockIO) string {
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
	result.WriteString(fmt.Sprintf("  %s\n", d.buildProgressBar(readPercentage, 35, fmt.Sprintf("Read %s", d.formatBytes(blockIO.ReadBytes)))))
	result.WriteString(fmt.Sprintf("  %s", d.buildProgressBar(writePercentage, 35, fmt.Sprintf("Write %s", d.formatBytes(blockIO.WriteBytes)))))

	return result.String()
}

func (d *Details) formatTime(t time.Time) string {
	if t.IsZero() {
		return "[gray]Never[white]"
	}
	return t.Format("2006-01-02 15:04:05")
}

func (d *Details) formatExitCode(code int) string {
	if code == 0 {
		return "[green]0 (success)[white]"
	} else if code > 0 {
		return fmt.Sprintf("[red]%d (error)[white]", code)
	}
	return "[gray]N/A[white]"
}

func (d *Details) formatPIDs(c *models.Container) string {
	if c.Stats != nil {
		return fmt.Sprintf("%d processes", c.Stats.PIDs)
	}
	return "[gray]N/A[white]"
}

func (d *Details) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (d *Details) formatNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	} else if num < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(num)/1000000000)
}

func (d *Details) getUsageColor(percentage float64) string {
	if percentage < 50 {
		return "green"
	} else if percentage < 80 {
		return "yellow"
	}
	return "red"
}

func (d *Details) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (d *Details) GetView() tview.Primitive {
	return d.view
}
