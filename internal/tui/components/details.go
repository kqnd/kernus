// internal/tui/components/details.go
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/kern/internal/models"
	"github.com/rivo/tview"
)

type Details struct {
	view             *tview.TextView
	currentContainer *models.Container
}

func NewDetails() *Details {
	d := &Details{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetWordWrap(true).
			SetScrollable(true),
	}

	d.view.SetBorder(true).SetTitle("Container Details")
	d.view.SetText("[yellow]Select a container to view details[white]")

	return d
}

func (d *Details) ShowContainer(container *models.Container) {
	d.currentContainer = container
	d.updateView()
}

func (d *Details) updateView() {
	if d.currentContainer == nil {
		d.view.SetText("[yellow]Select a container to view details[white]")
		return
	}

	content := d.buildContent()
	d.view.SetText(content)
}

func (d *Details) buildContent() string {
	c := d.currentContainer

	return fmt.Sprintf(`[yellow]Container: %s[white]

[yellow]Status:[white] [%s]%s %s[white]
[yellow]ID:[white] %s
[yellow]Image:[white] %s
[yellow]Command:[white] %s
[yellow]State:[white] %s

[yellow]Timing:[white]
• Created: %s
• Age: %s
• Uptime: %s

[yellow]Resources:[white]
• CPU: %.1f%%
• Memory: %s (%.1f%%)

[yellow]Networks (%d):[white]
%s

[yellow]Ports (%d):[white]
%s

[yellow]Mounts (%d):[white]
%s

[yellow]Labels (%d):[white]
%s`,
		c.ShortName(),
		c.Status.Color(), c.Status.Icon(), c.Status,
		c.ShortID(),
		c.Image,
		d.truncateString(c.Command, 50),
		c.State,
		d.formatTime(c.Created),
		c.FormatAge(),
		c.FormatUptime(),
		c.CPUUsage,
		c.Memory.String(), c.Memory.Percentage(),
		len(c.Networks),
		d.buildNetworksList(c.Networks),
		len(c.Ports),
		d.buildPortsList(c.Ports),
		len(c.Mounts),
		d.buildMountsList(c.Mounts),
		len(c.Labels),
		d.buildLabelsList(c.Labels))
}

func (d *Details) formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	duration := time.Since(t)
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(duration.Hours()))
}

func (d *Details) buildNetworksList(networks []models.Network) string {
	if len(networks) == 0 {
		return "• No networks"
	}

	var result strings.Builder
	for _, network := range networks {
		if network.IPAddress != "" {
			result.WriteString(fmt.Sprintf("• %s (%s)\n",
				network.Name, network.IPAddress))
		} else {
			result.WriteString(fmt.Sprintf("• %s\n", network.Name))
		}
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildPortsList(ports []models.Port) string {
	if len(ports) == 0 {
		return "• No ports exposed"
	}

	var result strings.Builder
	for _, port := range ports {
		result.WriteString(fmt.Sprintf("• %s\n", port.String()))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildMountsList(mounts []models.Mount) string {
	if len(mounts) == 0 {
		return "• No mounts"
	}

	var result strings.Builder
	for _, mount := range mounts {
		result.WriteString(fmt.Sprintf("• %s (%s, %s)\n",
			mount.String(), mount.Type, mount.Mode))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (d *Details) buildLabelsList(labels map[string]string) string {
	if len(labels) == 0 {
		return "• No labels"
	}

	var result strings.Builder
	count := 0
	maxLabels := 10 // Limita a exibição para não ficar muito longo

	for key, value := range labels {
		if count >= maxLabels {
			result.WriteString("• ... and more\n")
			break
		}

		truncatedValue := d.truncateString(value, 30)
		result.WriteString(fmt.Sprintf("• %s: %s\n", key, truncatedValue))
		count++
	}

	return strings.TrimSuffix(result.String(), "\n")
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
