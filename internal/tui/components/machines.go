package components

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kqnd/kernus/internal/models"
	"github.com/rivo/tview"
)

type MachineList struct {
	list       *tview.List
	machines   []*models.Machine
	onSelected func(*models.Machine)
}

func NewMachineList(machines []*models.Machine) *MachineList {
	ml := &MachineList{
		list:     tview.NewList(),
		machines: machines,
	}

	ml.setupView()
	ml.setupKeyBindings()
	ml.refreshView()
	return ml
}

func (ml *MachineList) setupView() {
	ml.list.SetBorder(true).
		SetTitle("Machines")

	ml.list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shorcut rune) {
		if index < len(ml.machines) && ml.onSelected != nil {
			ml.onSelected(ml.machines[index])
		}
	})
}

func (ml *MachineList) setupKeyBindings() {
	ml.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		}
		return event
	})
}

func (ml *MachineList) SetSelectedFunc(fn func(*models.Machine)) {
	ml.onSelected = fn
}

func (ml *MachineList) UpdateMachines(machines []*models.Machine) {
	ml.machines = machines
	ml.refreshView()
}

func (ml *MachineList) refreshView() {
	ml.list.Clear()

	title := ml.buildTitle()
	ml.list.SetTitle(title)

	for i, machine := range ml.machines {
		mainText := machine.Name
		secondaryText := ml.formatSecondaryText(machine)

		ml.list.AddItem(mainText, secondaryText, rune('0'+i%10), nil)
	}
}

func (ml *MachineList) buildTitle() string {
	title := "Machines"
	return title
}

func (ml *MachineList) formatSecondaryText(machine *models.Machine) string {
	statusColor := machine.Status.Color()

	if machine.Status == models.StatusOffline {
		timeSince := time.Since(machine.LastSeen)
		return fmt.Sprintf("[%s]%s[white] | Last: %s", statusColor, machine.Status, ml.formatDuration(timeSince))
	}

	return fmt.Sprintf("[%s]%s[white]  | CPU: %.1f%%", statusColor, machine.Status, machine.CPUUsage)
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

func (ml *MachineList) GetSelectedMachine() *models.Machine {
	index := ml.list.GetCurrentItem()
	if index >= 0 && index < len(ml.machines) {
		return ml.machines[index]
	}
	return nil
}

func (ml *MachineList) SelectMachine(id string) {
	for i, machine := range ml.machines {
		if machine.ID == id {
			ml.list.SetCurrentItem(i)
			break
		}
	}
}

func (ml *MachineList) GetMachineCount() int {
	return len(ml.machines)
}

func (ml *MachineList) GetView() tview.Primitive {
	return ml.list
}
