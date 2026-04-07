package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kiev/kernus/internal/models"
	"github.com/kiev/kernus/internal/tui/components/details"
	"github.com/rivo/tview"
)

type Details struct {
	View        *tview.TextView
	currentTab  int
	tabNames    []string
	container   *models.Container
	machine     *models.Machine
	machineMode bool
	onboarding  string
	// lastViewKey / lastViewTab detect refresh of the same entity+tab so we can preserve scroll.
	lastViewKey string
	lastViewTab int
}

func NewDetails(machineMode bool) *Details {
	d := &Details{
		View:        tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		machineMode: machineMode,
	}

	if machineMode {
		d.tabNames = []string{"Overview", "Resources", "Processes"}
	} else {
		d.tabNames = []string{"Overview", "Stats", "Network", "Storage", "Logs"}
	}

	d.View.SetBorder(false)
	d.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			if d.currentTab > 0 {
				d.currentTab--
				d.render()
			}
			return nil
		case tcell.KeyRight:
			if d.currentTab < len(d.tabNames)-1 {
				d.currentTab++
				d.render()
			}
			return nil
		}
		return event
	})

	d.render()
	return d
}

func (d *Details) SetTab(tab int) {
	if tab >= 0 && tab < len(d.tabNames) {
		d.currentTab = tab
		d.render()
	}
}

func (d *Details) UpdateContainer(c *models.Container) {
	d.container = c
	d.machine = nil
	d.render()
}

func (d *Details) SetOnboardingMessage(message string) {
	d.onboarding = message
	d.render()
}

func (d *Details) UpdateMachine(m *models.Machine) {
	d.machine = m
	d.container = nil
	d.render()
}

func (d *Details) render() {
	savedRow, savedCol := d.View.GetScrollOffset()

	var currentKey string
	if d.machineMode {
		if d.machine != nil {
			currentKey = "m:" + d.machine.Name
		}
	} else if d.container != nil {
		currentKey = "c:" + d.container.ID
	}
	wasSameView := currentKey != "" && currentKey == d.lastViewKey && d.currentTab == d.lastViewTab

	var b strings.Builder

	b.WriteString(d.renderTabs())
	b.WriteString("\n")

	if d.machineMode {
		if d.machine == nil {
			if strings.TrimSpace(d.onboarding) != "" {
				b.WriteString("\n")
				b.WriteString(d.onboarding)
			} else {
				b.WriteString("\n  [gray]Select a machine to view details[white]")
			}
			d.View.SetText(b.String())
			d.View.ScrollToBeginning()
			d.lastViewKey = ""
			d.lastViewTab = d.currentTab
			return
		}
		d.renderMachineContent(&b)
	} else {
		if d.container == nil {
			if strings.TrimSpace(d.onboarding) != "" {
				b.WriteString("\n")
				b.WriteString(d.onboarding)
			} else {
				b.WriteString("\n  [gray]Select a container to view details[white]")
			}
			d.View.SetText(b.String())
			d.View.ScrollToBeginning()
			d.lastViewKey = ""
			d.lastViewTab = d.currentTab
			return
		}
		d.renderContainerContent(&b)
	}

	d.View.SetText(b.String())

	// Logs tab: keep tail in view as new lines arrive (ScrollToEnd enables trackEnd in tview).
	if !d.machineMode && d.currentTab == 4 {
		d.View.ScrollToEnd()
	} else if wasSameView {
		d.View.ScrollTo(savedRow, savedCol)
	} else {
		d.View.ScrollToBeginning()
	}

	d.lastViewKey = currentKey
	d.lastViewTab = d.currentTab
}

func (d *Details) renderTabs() string {
	var parts []string
	for i, name := range d.tabNames {
		if i == d.currentTab {
			parts = append(parts, fmt.Sprintf(" [white::b]> %s <[-:-:-]", name))
		} else {
			parts = append(parts, fmt.Sprintf(" [gray]%s[white]", name))
		}
	}
	return " " + strings.Join(parts, " ")
}

func (d *Details) renderContainerContent(b *strings.Builder) {
	switch d.currentTab {
	case 0:
		details.RenderOverview(b, d.container)
	case 1:
		details.RenderStats(b, d.container)
	case 2:
		details.RenderNetwork(b, d.container)
	case 3:
		details.RenderStorage(b, d.container)
	case 4:
		details.RenderLogs(b, d.container)
	}
}

func (d *Details) renderMachineContent(b *strings.Builder) {
	switch d.currentTab {
	case 0:
		d.renderMachineOverview(b)
	case 1:
		d.renderMachineResources(b)
	case 2:
		d.renderMachineProcesses(b)
	}
}

func (d *Details) renderMachineOverview(b *strings.Builder) {
	m := d.machine

	b.WriteString("\n  [white::b]Machine Information[white:-:-]\n\n")
	b.WriteString("  [gray]Identity[white]\n")
	fmt.Fprintf(b, "    Name     : %s\n", m.Name)
	fmt.Fprintf(b, "    IP       : %s\n", m.IP)
	fmt.Fprintf(b, "    Group    : %s\n", m.Group)

	statusColor := m.Status.Color()
	icon := "●"
	switch m.Status {
	case models.MachineOffline:
		icon = "○"
	case models.MachineError:
		icon = "◐"
	}
	fmt.Fprintf(b, "    Status   : [%s]%s %s[white]\n", statusColor, icon, m.Status)

	b.WriteString("\n  [gray]Resources[white]\n")
	fmt.Fprintf(b, "    CPU      : %s %.1f%%\n", details.BuildBar(m.CPUUsage, 30), m.CPUUsage)
	memPct := m.MemoryUsage.Percentage()
	fmt.Fprintf(b, "    Memory   : %s %.1f%%\n", details.BuildBar(memPct, 30), memPct)
	fmt.Fprintf(b, "               %s\n", m.MemoryUsage.String())
	diskPct := m.DiskUsage.Percentage()
	fmt.Fprintf(b, "    Disk     : %s %.1f%%\n", details.BuildBar(diskPct, 30), diskPct)
	fmt.Fprintf(b, "               %s\n", m.DiskUsage.String())

	b.WriteString("\n")
	fmt.Fprintf(b, "  Uptime     : %s\n", m.Uptime.String())

	if m.Status != models.MachineOnline {
		ago := formatTimeSince(time.Since(m.LastSeen))
		fmt.Fprintf(b, "  Last Seen  : %s ago\n", ago)
	} else {
		fmt.Fprintf(b, "  Last Seen  : just now\n")
	}
}

func (d *Details) renderMachineResources(b *strings.Builder) {
	m := d.machine

	b.WriteString("\n  [white::b]CPU Usage[white:-:-]\n")
	fmt.Fprintf(b, "    %s %.1f%%\n", details.BuildBar(m.CPUUsage, 40), m.CPUUsage)

	b.WriteString("\n  [white::b]Memory Usage[white:-:-]\n")
	memPct := m.MemoryUsage.Percentage()
	fmt.Fprintf(b, "    Total: %s %.1f%%\n", details.BuildBar(memPct, 40), memPct)
	fmt.Fprintf(b, "    %s\n", m.MemoryUsage.String())

	b.WriteString("\n  [white::b]Disk Usage[white:-:-]\n")
	diskPct := m.DiskUsage.Percentage()
	fmt.Fprintf(b, "    Total: %s %.1f%%\n", details.BuildBar(diskPct, 40), diskPct)
	fmt.Fprintf(b, "    %s\n", m.DiskUsage.String())
}

func (d *Details) renderMachineProcesses(b *strings.Builder) {
	m := d.machine

	b.WriteString("\n  [white::b]Listening Processes[white:-:-]\n\n")

	if len(m.Processes) == 0 {
		b.WriteString("  [gray]No processes information available[white]\n")
		return
	}

	b.WriteString("  ┌────────────────────┬────────┬──────────────────────┐\n")
	b.WriteString("  │ Address            │ Port   │ Name                 │\n")
	b.WriteString("  ├────────────────────┼────────┼──────────────────────┤\n")
	for _, p := range m.Processes {
		fmt.Fprintf(b, "  │ %-18s │ %-6d │ %-20s │\n", p.Address, p.Port, p.Name)
	}
	b.WriteString("  └────────────────────┴────────┴──────────────────────┘\n")
}
