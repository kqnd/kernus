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
	view           *tview.TextView
	currentMachine *models.Machine
}

func NewDetails() *Details {
	d := &Details{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetWordWrap(true).
			SetScrollable(true),
	}

	d.view.SetBorder(true).SetTitle("Machine Details")
	d.view.SetText("[yellow]Select a machine to view details[white]")

	return d
}

func (d *Details) ShowMachine(machine *models.Machine) {
	d.currentMachine = machine
	d.updateView()
}

func (d *Details) updateView() {
	if d.currentMachine == nil {
		d.view.SetText("[yellow]Select a machine to view details[white]")
		return
	}

	content := d.buildContent()
	d.view.SetText(content)
}

func (d *Details) buildContent() string {
	m := d.currentMachine

	return fmt.Sprintf(`[yellow]Machine: %s[white]

[yellow]Status:[white] [%s]%s[white]
[yellow]IP:[white] %s
[yellow]Group:[white] %s
[yellow]Last Seen:[white] %s

[yellow]Resources:[white]
• CPU: %.1f%%
• Memory: %s (%.1f%%)
• Disk: %s (%.1f%%)
• Uptime: %s

[yellow]Services (%d):[white]
%s`,
		m.Name,
		m.Status.Color(), m.Status,
		m.IP,
		m.Group,
		d.formatTime(m.LastSeen),
		m.CPUUsage,
		m.MemoryUsage.String(), m.MemoryUsage.Percentage(),
		m.DiskUsage.String(), m.DiskUsage.Percentage(),
		m.Uptime.String(),
		len(m.Processes),
		d.buildServicesList(m.Processes))
}

func (d *Details) formatTime(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(duration.Hours()))
}

func (d *Details) buildServicesList(processes []models.Process) string {
	if len(processes) == 0 {
		return "• No services running"
	}

	var services strings.Builder
	for _, proc := range processes {
		services.WriteString(fmt.Sprintf("• %s (%s:%d)\n",
			proc.Name, proc.Address, proc.Port))
	}

	return strings.TrimSuffix(services.String(), "\n")
}

func (d *Details) GetView() tview.Primitive {
	return d.view
}
