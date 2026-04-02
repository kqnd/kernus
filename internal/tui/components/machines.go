package components

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kiev/kernus/internal/models"
	"github.com/rivo/tview"
)

type MachineList struct {
	View         *tview.List
	machines     []*models.Machine
	groups       map[string][]*models.Machine
	groupOrder   []string
	expanded     map[string]bool
	items        []machineListItem
	selectedFunc func(*models.Machine)
}

type machineListItem struct {
	isGroup   bool
	groupName string
	machine   *models.Machine
}

func NewMachineList() *MachineList {
	ml := &MachineList{
		View:     tview.NewList(),
		expanded: make(map[string]bool),
	}

	ml.View.SetBorder(false)
	ml.View.SetTitle(" Machines ")
	ml.View.SetHighlightFullLine(true)
	ml.View.ShowSecondaryText(false)
	ml.View.SetSelectedBackgroundColor(tcell.ColorDarkBlue)

	ml.View.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(ml.items) {
			item := ml.items[index]
			if !item.isGroup && item.machine != nil && ml.selectedFunc != nil {
				ml.selectedFunc(item.machine)
			}
		}
	})

	ml.View.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(ml.items) {
			item := ml.items[index]
			if item.isGroup {
				ml.expanded[item.groupName] = !ml.expanded[item.groupName]
				ml.rebuildList()
			}
		}
	})

	ml.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		idx := ml.View.GetCurrentItem()
		if idx < 0 || idx >= len(ml.items) {
			return event
		}
		item := ml.items[idx]

		switch event.Key() {
		case tcell.KeyRight:
			if item.isGroup && !ml.expanded[item.groupName] {
				ml.expanded[item.groupName] = true
				ml.rebuildList()
				return nil
			}
		case tcell.KeyLeft:
			if item.isGroup && ml.expanded[item.groupName] {
				ml.expanded[item.groupName] = false
				ml.rebuildList()
				return nil
			}
		}
		return event
	})

	return ml
}

func (ml *MachineList) SetSelectedFunc(fn func(*models.Machine)) {
	ml.selectedFunc = fn
}

func (ml *MachineList) GetSelectedMachine() *models.Machine {
	idx := ml.View.GetCurrentItem()
	if idx < 0 || idx >= len(ml.items) {
		return nil
	}
	item := ml.items[idx]
	if item.isGroup {
		return nil
	}
	return item.machine
}

func (ml *MachineList) UpdateMachines(machines []*models.Machine) {
	ml.machines = machines
	ml.buildGroups()
	ml.rebuildList()
}

func (ml *MachineList) buildGroups() {
	ml.groups = make(map[string][]*models.Machine)
	for _, m := range ml.machines {
		group := m.Group
		if group == "" {
			group = "default"
		}
		ml.groups[group] = append(ml.groups[group], m)
	}

	ml.groupOrder = make([]string, 0, len(ml.groups))
	for g := range ml.groups {
		ml.groupOrder = append(ml.groupOrder, g)
	}
	sort.Strings(ml.groupOrder)

	for g := range ml.groups {
		if _, exists := ml.expanded[g]; !exists {
			ml.expanded[g] = true
		}
	}
}

func (ml *MachineList) rebuildList() {
	ml.View.Clear()
	ml.items = nil

	total := len(ml.machines)
	ml.View.SetTitle(fmt.Sprintf(" Machines (%d total) ", total))

	for _, groupName := range ml.groupOrder {
		members := ml.groups[groupName]

		expanded := ml.expanded[groupName]
		folderIcon := "📁"
		if expanded {
			folderIcon = "📂"
		}

		groupText := fmt.Sprintf("%s [yellow]%s[white] (%d)", folderIcon, groupName, len(members))
		ml.View.AddItem(groupText, "", 0, nil)
		ml.items = append(ml.items, machineListItem{isGroup: true, groupName: groupName})

		if expanded {
			for _, m := range members {
				icon := "●"
				color := m.Status.Color()
				switch m.Status {
				case models.MachineOffline:
					icon = "○"
				case models.MachineError:
					icon = "◐"
				}

				info := ""
				if m.Status == models.MachineOnline {
					info = fmt.Sprintf("  CPU: %.1f%%", m.CPUUsage)
				} else if m.Status == models.MachineOffline {
					ago := time.Since(m.LastSeen)
					info = fmt.Sprintf("  %s ago", formatTimeSince(ago))
				}

				text := fmt.Sprintf("  [%s]%s %s[white]%s", color, icon, m.Name, info)
				ml.View.AddItem(text, "", 0, nil)
				ml.items = append(ml.items, machineListItem{machine: m})
			}
		}
	}
}

func formatTimeSince(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
