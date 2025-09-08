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

type MachineList struct {
	list       *tview.List
	machines   []*models.Machine
	filtered   []*models.Machine
	onSelected func(*models.Machine)
	sortBy     SortType
	ascending  bool
	filter     string
}

type SortType int

const (
	SortByName SortType = iota
	SortByStatus
	SortByCPU
	SortByMemory
	SortByLastSeen
)

func NewMachineList() *MachineList {
	ml := &MachineList{
		list:      tview.NewList(),
		machines:  make([]*models.Machine, 0),
		filtered:  make([]*models.Machine, 0),
		sortBy:    SortByName,
		ascending: true,
	}

	ml.setupView()
	ml.setupKeyBindings()
	return ml
}

func (ml *MachineList) setupView() {
	ml.list.SetBorder(true).
		SetTitle("Machines [Enter=Select, S=Sort, F=Filter]")

	ml.list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shorcut rune) {
		if index < len(ml.filtered) && ml.onSelected != nil {
			ml.onSelected(ml.filtered[index])
		}
	})
}

func (ml *MachineList) setupKeyBindings() {
	ml.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case 's', 'S':
			ml.cycleSortType()
			return nil
		case 'f', 'F':
			ml.showFilterDialog()
			return nil
		case 'c', 'C':
			ml.clearFilter()
			return nil
		case 'r', 'R':
			ml.toggleSortOrder()
			return nil
		}
		return event
	})
}

func (ml *MachineList) SetSelectedFunc(fn func(*models.Machine)) {
	ml.onSelected = fn
}

func (ml *MachineList) UpdateMachines(machines []*models.Machine) {
	ml.machines = machines
	ml.applyFilterAndSort()
	ml.refreshView()
}

func (ml *MachineList) applyFilterAndSort() {
	if ml.filter == "" {
		ml.filtered = make([]*models.Machine, len(ml.machines))
		copy(ml.filtered, ml.machines)
	} else {
		ml.filtered = make([]*models.Machine, 0)
		filterLower := strings.ToLower(ml.filter)

		for _, machine := range ml.machines {
			if strings.Contains(strings.ToLower(machine.Name), filterLower) ||
				strings.Contains(strings.ToLower(string(machine.Status)), filterLower) ||
				strings.Contains(strings.ToLower(machine.Group), filterLower) ||
				strings.Contains(strings.ToLower(machine.IP), filterLower) {
				ml.filtered = append(ml.filtered, machine)
			}
		}
	}

	ml.sortMachines()
}

func (ml *MachineList) sortMachines() {
	sort.Slice(ml.filtered, func(i, j int) bool {
		var less bool
		switch ml.sortBy {
		case SortByName:
			less = ml.filtered[i].Name < ml.filtered[j].Name
		case SortByStatus:
			less = ml.filtered[i].Status < ml.filtered[j].Status
		case SortByCPU:
			less = ml.filtered[i].CPUUsage < ml.filtered[j].CPUUsage
		case SortByMemory:
			less = ml.filtered[i].MemoryUsage.Percentage() < ml.filtered[j].MemoryUsage.Percentage()
		case SortByLastSeen:
			less = ml.filtered[i].LastSeen.Before(ml.filtered[j].LastSeen)
		}

		if ml.ascending {
			return less
		}
		return !less
	})
}

func (ml *MachineList) refreshView() {
	ml.list.Clear()

	title := ml.buildTitle()
	ml.list.SetTitle(title)

	for i, machine := range ml.filtered {
		mainText := machine.Name
		secondaryText := ml.formatSecondaryText(machine)

		ml.list.AddItem(mainText, secondaryText, rune('0'+i%10), nil)
	}
}

func (ml *MachineList) buildTitle() string {
	title := "Machines"

	sortNames := []string{"Name", "Status", "CPU", "Memory", "LastSeen"}
	direction := "↑"
	if !ml.ascending {
		direction = "↓"
	}
	title += fmt.Sprintf(" [Sort: %s%s]", sortNames[ml.sortBy], direction)
	if ml.filter != "" {
		title += fmt.Sprintf(" [Filter: %s]", ml.filter)
	}

	total := len(ml.machines)
	filtered := len(ml.filtered)
	if filtered != total {
		title += fmt.Sprintf(" [%d/%d]", filtered, total)
	} else {
		title += fmt.Sprintf(" [%d]", total)
	}

	return title
}

func (ml *MachineList) formatSecondaryText(machine *models.Machine) string {
	statusColor := machine.Status.Color()

	if machine.Status == models.StatusOffline {
		timeSince := time.Since(machine.LastSeen)
		return fmt.Sprintf("[%s]%s[white] | Last: %s | Group: %s", statusColor, machine.Status, ml.formatDuration(timeSince), machine.Group)
	}

	return fmt.Sprintf("[%s]%s[white] | CPU: %.1f%% | Mem: %.1f%% | %s", statusColor, machine.Status, machine.CPUUsage, machine.MemoryUsage.Percentage(), machine.Group)
}

func (ml *MachineList) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func (ml *MachineList) cycleSortType() {
	ml.sortBy = SortType((int(ml.sortBy) + 1) % 5)
	ml.applyFilterAndSort()
	ml.refreshView()
}

func (ml *MachineList) toggleSortOrder() {
	ml.ascending = !ml.ascending
	ml.applyFilterAndSort()
	ml.refreshView()
}

func (ml *MachineList) showFilterDialog() {
	ml.cycleStatusFilter()
}

func (ml *MachineList) cycleStatusFilter() {
	filters := []string{"", "online", "offline", "warning"}
	currentIndex := 0

	for i, filter := range filters {
		if filter == ml.filter {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(filters)
	ml.filter = filters[nextIndex]

	ml.applyFilterAndSort()
	ml.refreshView()
}

func (ml *MachineList) clearFilter() {
	ml.filter = ""
	ml.applyFilterAndSort()
	ml.refreshView()
}

func (ml *MachineList) GetSelectedMachine() *models.Machine {
	index := ml.list.GetCurrentItem()
	if index >= 0 && index < len(ml.filtered) {
		return ml.filtered[index]
	}
	return nil
}

func (ml *MachineList) SelectMachine(id string) {
	for i, machine := range ml.filtered {
		if machine.ID == id {
			ml.list.SetCurrentItem(i)
			break
		}
	}
}

func (ml *MachineList) GetMachineCount() int {
	return len(ml.filtered)
}

func (ml *MachineList) GetView() tview.Primitive {
	return ml.list
}
