package details

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kern/internal/models"
)

type OverviewTab struct {
	formatter *Formatter
}

func NewOverviewTab() *OverviewTab {
	return &OverviewTab{
		formatter: NewFormatter(),
	}
}

func (o *OverviewTab) Render(container *models.Container) string {
	if container == nil {
		return ""
	}

	sections := []string{
		o.renderIdentitySection(container),
		o.renderStatusSection(container),
		o.renderTimingSection(container),
		o.renderConfigSection(container),
		o.renderQuickStats(container),
		o.renderLabels(container),
	}

	return fmt.Sprintf("[yellow]Container Information[white]\n\n%s", strings.Join(sections, "\n\n"))
}

func (o *OverviewTab) renderIdentitySection(c *models.Container) string {
	return fmt.Sprintf(`[yellow]Identity[white]
	ID       : %s
	Name     : %s
	Image    : [cyan]%s[white]
	Tag      : [blue]%s[white]`,
		c.ShortID(),
		c.ShortName(),
		c.ImageName(),
		c.ImageTag())
}

func (o *OverviewTab) renderStatusSection(c *models.Container) string {
	healthStatus := "Unknown"
	healthColor := "gray"
	healthIcon := ""

	if c.Health != nil {
		healthStatus = string(c.Health.Status)
		healthColor = c.Health.Status.Color()
		healthIcon = c.Health.Status.Icon()
	}

	return fmt.Sprintf(`[yellow]Status[white]
	Status   : [%s]%s %s[white]
	State    : %s
	Health   : [%s]%s %s[white]
	Exit Code: %s`,
		c.Status.Color(), c.Status.Icon(), c.Status,
		c.State,
		healthColor, healthIcon, healthStatus,
		o.formatter.FormatExitCode(c.ExitCode))
}

func (o *OverviewTab) renderTimingSection(c *models.Container) string {
	return fmt.Sprintf(`[yellow]Timing[white]
	Created  : %s
	Started  : %s
	Age      : %s
	Uptime   : %s`,
		o.formatter.FormatTime(c.Created),
		o.formatter.FormatTime(c.Started),
		c.FormatAge(),
		c.FormatUptime())
}

func (o *OverviewTab) renderConfigSection(c *models.Container) string {
	restartPolicy := "No restart"
	if c.RestartPolicy.Name != "" {
		restartPolicy = c.RestartPolicy.Name
		if c.RestartPolicy.MaximumRetryCount > 0 {
			restartPolicy = fmt.Sprintf("%s (max: %d)", restartPolicy, c.RestartPolicy.MaximumRetryCount)
		}
	}

	return fmt.Sprintf(`[yellow]Configuration[white]
	Command  : [cyan]%s[white]
	Restart  : %s
	PIDs     : %s`,
		o.formatter.TruncateString(c.Command, 60),
		restartPolicy,
		o.formatPIDs(c))
}

func (o *OverviewTab) renderQuickStats(c *models.Container) string {
	if c.Stats == nil {
		return `[yellow]Quick Stats[white]
	CPU      : [gray]N/A (not running)[white]
	Memory   : [gray]N/A (not running)[white]
	Network  : [gray]N/A (not running)[white]
	PIDs     : [gray]N/A (not running)[white]`
	}

	cpuColor := o.formatter.GetUsageColor(c.Stats.CPU.Usage)
	memColor := o.formatter.GetUsageColor(c.Stats.Memory.Percentage())

	return fmt.Sprintf(`[yellow]Quick Stats[white]
	CPU      : [%s]%.1f%%[white]
	Memory   : [%s]%.1fMB (%.1f%%)[white]
	Network  : ↓ %s ↑ %s
	PIDs     : %d processes`,
		cpuColor, c.Stats.CPU.Usage,
		memColor, float64(c.Stats.Memory.Usage)/1024/1024, c.Stats.Memory.Percentage(),
		o.formatter.FormatBytes(c.Stats.Network.RxBytes),
		o.formatter.FormatBytes(c.Stats.Network.TxBytes),
		c.Stats.PIDs)
}

func (o *OverviewTab) renderLabels(c *models.Container) string {
	labelsGrid := o.buildLabelsGrid(c.Labels)
	return fmt.Sprintf("[yellow]Labels (%d)[white]\n%s", len(c.Labels), labelsGrid)
}

func (o *OverviewTab) buildLabelsGrid(labels map[string]string) string {
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
		truncatedKey := o.formatter.TruncateString(key, 20)
		truncatedValue := o.formatter.TruncateString(value, 30)

		result.WriteString(fmt.Sprintf("  [yellow]%s[white]: %s\n", truncatedKey, truncatedValue))
		count++
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (o *OverviewTab) formatPIDs(c *models.Container) string {
	if c.Stats != nil {
		return fmt.Sprintf("%d processes", c.Stats.PIDs)
	}
	return "[gray]N/A[white]"
}
