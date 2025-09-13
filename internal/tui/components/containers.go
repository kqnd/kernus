package components

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/kern/internal/models"
	"github.com/rivo/tview"
)

type ContainerList struct {
	list       *tview.List
	containers []*models.Container
	onSelected func(*models.Container)
}

func NewContainerList(containers []*models.Container) *ContainerList {
	cl := &ContainerList{
		list:       tview.NewList(),
		containers: containers,
	}

	cl.setupView()
	cl.setupKeyBindings()
	cl.refreshView()
	return cl
}

func (cl *ContainerList) UpdateContainersPreserveSelection(containers []*models.Container, selectedID string) {
	cl.containers = containers

	cl.refreshView()

	if selectedID != "" {
		cl.SelectContainer(selectedID)
	} else if len(cl.containers) > 0 {
		cl.list.SetCurrentItem(0)
		if cl.onSelected != nil {
			cl.onSelected(cl.containers[0])
		}
	}
}

func (cl *ContainerList) setupView() {
	cl.list.SetBorder(true).
		SetTitle("Containers")

	cl.list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index < len(cl.containers) && cl.onSelected != nil {
			cl.onSelected(cl.containers[index])
		}
	})
}

func (cl *ContainerList) setupKeyBindings() {
	cl.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			index := cl.list.GetCurrentItem()
			if index >= 0 && index < len(cl.containers) && cl.onSelected != nil {
				cl.onSelected(cl.containers[index])
			}
			return nil
		}
		return event
	})
}

func (cl *ContainerList) SetSelectedFunc(fn func(*models.Container)) {
	cl.onSelected = fn
}

func (cl *ContainerList) UpdateContainers(containers []*models.Container) {
	cl.containers = containers
	cl.refreshView()
}

func (cl *ContainerList) refreshView() {
	cl.list.Clear()

	title := cl.buildTitle()
	cl.list.SetTitle(title)

	for i, container := range cl.containers {
		mainText := cl.formatMainText(container)
		secondaryText := cl.formatSecondaryText(container)

		cl.list.AddItem(mainText, secondaryText, rune('0'+i%10), nil)
	}
}

func (cl *ContainerList) buildTitle() string {
	runningCount := 0
	stoppedCount := 0
	pausedCount := 0

	for _, container := range cl.containers {
		switch container.Status {
		case models.StatusRunning:
			runningCount++
		case models.StatusExited, models.StatusStopped, models.StatusDead:
			stoppedCount++
		case models.StatusPaused:
			pausedCount++
		}
	}

	title := fmt.Sprintf("Containers (%d total", len(cl.containers))
	title += ")"

	return title
}

func (cl *ContainerList) formatMainText(container *models.Container) string {
	statusColor := container.Status.Color()

	return fmt.Sprintf("[%s]%s[white] (%s)",
		statusColor,
		container.ShortName(),
		container.ShortID())
}

func (cl *ContainerList) formatSecondaryText(container *models.Container) string {
	statusColor := container.Status.Color()

	switch container.Status {
	case models.StatusRunning:
		return fmt.Sprintf("[%s]%s[white] Port: %s",
			statusColor,
			container.Status,
			container.ShortPort())

	case models.StatusExited, models.StatusStopped, models.StatusDead:
		return fmt.Sprintf("[%s]%s[white] Age: %s",
			statusColor,
			container.Status,
			container.FormatAge())

	case models.StatusPaused:
		return fmt.Sprintf("[%s]%s[white] Port: %s",
			statusColor,
			container.Status,
			container.ShortPort())

	default:
		return fmt.Sprintf("[%s]%s[white] Age: %s",
			statusColor,
			container.Status,
			container.FormatAge(),
		)
	}
}

func (cl *ContainerList) GetSelectedContainer() *models.Container {
	index := cl.list.GetCurrentItem()
	if index >= 0 && index < len(cl.containers) {
		return cl.containers[index]
	}
	return nil
}

func (cl *ContainerList) SelectContainer(id string) {
	for i, container := range cl.containers {
		if container.ID == id {
			cl.list.SetCurrentItem(i)
			break
		}
	}
}

func (cl *ContainerList) GetContainerCount() int {
	return len(cl.containers)
}

func (cl *ContainerList) GetView() tview.Primitive {
	return cl.list
}
