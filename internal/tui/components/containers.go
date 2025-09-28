package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kqnd/kernus/internal/models"
	"github.com/rivo/tview"
)

type ContainerGroup struct {
	Name       string
	Containers []*models.Container
	IsExpanded bool
}

type ContainerItem struct {
	Container *models.Container
	Group     *ContainerGroup
	IsGroup   bool
	Level     int
}

type ContainerList struct {
	list         *tview.List
	containers   []*models.Container
	groups       []*ContainerGroup
	displayItems []*ContainerItem
	onSelected   func(*models.Container)
}

func NewContainerList(containers []*models.Container) *ContainerList {
	cl := &ContainerList{
		list:       tview.NewList(),
		containers: containers,
		groups:     make([]*ContainerGroup, 0),
	}

	cl.setupView()
	cl.setupKeyBindings()
	cl.buildGroups()
	cl.refreshView()
	return cl
}

func (cl *ContainerList) UpdateContainersPreserveSelection(containers []*models.Container, selectedID string) {
	cl.containers = containers
	cl.buildGroups()
	cl.refreshView()

	if selectedID != "" {
		cl.SelectContainer(selectedID)
	} else if len(cl.displayItems) > 0 {
		cl.list.SetCurrentItem(0)
		cl.handleItemSelection(0)
	}
}

func (cl *ContainerList) setupView() {
	cl.list.SetBorder(true).
		SetTitle("Containers")

	cl.list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		cl.handleItemSelection(index)
	})
}

func (cl *ContainerList) setupKeyBindings() {
	cl.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			index := cl.list.GetCurrentItem()
			cl.handleItemSelection(index)
			return nil
		case tcell.KeyRight:
			index := cl.list.GetCurrentItem()
			if index >= 0 && index < len(cl.displayItems) {
				item := cl.displayItems[index]
				if item.IsGroup && !item.Group.IsExpanded {
					item.Group.IsExpanded = true
					cl.refreshView()
				}
			}
			return nil
		case tcell.KeyLeft:
			index := cl.list.GetCurrentItem()
			if index >= 0 && index < len(cl.displayItems) {
				item := cl.displayItems[index]
				if item.IsGroup && item.Group.IsExpanded {
					item.Group.IsExpanded = false
					cl.refreshView()
				} else if !item.IsGroup && item.Level > 0 {
					cl.selectParentGroup(item)
				}
			}
			return nil
		}
		return event
	})
}

func (cl *ContainerList) handleItemSelection(index int) {
	if index < 0 || index >= len(cl.displayItems) {
		return
	}

	item := cl.displayItems[index]
	if item.IsGroup {
		item.Group.IsExpanded = !item.Group.IsExpanded
		cl.refreshView()
	} else if item.Container != nil && cl.onSelected != nil {
		cl.onSelected(item.Container)
	}
}

func (cl *ContainerList) selectParentGroup(item *ContainerItem) {
	if item.Group == nil {
		return
	}

	for i, displayItem := range cl.displayItems {
		if displayItem.IsGroup && displayItem.Group == item.Group {
			cl.list.SetCurrentItem(i)
			break
		}
	}
}

func (cl *ContainerList) buildGroups() {
	cl.groups = make([]*ContainerGroup, 0)

	prefixMap := make(map[string][]*models.Container)
	usedContainers := make(map[string]bool)

	for _, container := range cl.containers {
		if usedContainers[container.ID] {
			continue
		}

		name := container.ShortName()
		prefix := cl.extractPrefix(name)

		if prefix != name {
			groupContainers := []*models.Container{}
			for _, c := range cl.containers {
				if strings.HasPrefix(c.ShortName(), prefix) && !usedContainers[c.ID] {
					groupContainers = append(groupContainers, c)
					usedContainers[c.ID] = true
				}
			}

			if len(groupContainers) > 1 {
				prefixMap[prefix] = groupContainers
			} else if len(groupContainers) == 1 {
				usedContainers[groupContainers[0].ID] = false
			}
		}
	}

	for _, container := range cl.containers {
		if !usedContainers[container.ID] {
			prefixMap[container.ShortName()] = []*models.Container{container}
		}
	}

	for prefix, containers := range prefixMap {
		group := &ContainerGroup{
			Name:       prefix,
			Containers: containers,
			IsExpanded: len(containers) == 1,
		}
		cl.groups = append(cl.groups, group)
	}
}

func (cl *ContainerList) extractPrefix(name string) string {

	parts := strings.Split(name, "-")
	if len(parts) <= 1 {
		return name
	}

	for i := len(parts) - 1; i > 0; i-- {
		candidatePrefix := strings.Join(parts[:i], "-")

		matchCount := 0
		for _, container := range cl.containers {
			containerName := container.ShortName()
			if strings.HasPrefix(containerName, candidatePrefix) && containerName != name {
				matchCount++
				if matchCount >= 1 {
					return candidatePrefix
				}
			}
		}
	}

	return name
}

func (cl *ContainerList) refreshView() {
	cl.list.Clear()
	cl.displayItems = make([]*ContainerItem, 0)

	title := cl.buildTitle()
	cl.list.SetTitle(title)

	for _, group := range cl.groups {
		if len(group.Containers) > 1 {
			groupItem := &ContainerItem{
				Group:   group,
				IsGroup: true,
				Level:   0,
			}
			cl.displayItems = append(cl.displayItems, groupItem)

			mainText := cl.formatGroupText(group)
			secondaryText := cl.formatGroupSecondary(group)
			cl.list.AddItem(mainText, secondaryText, 0, nil)

			if group.IsExpanded {
				for _, container := range group.Containers {
					containerItem := &ContainerItem{
						Container: container,
						Group:     group,
						IsGroup:   false,
						Level:     1,
					}
					cl.displayItems = append(cl.displayItems, containerItem)

					mainText := cl.formatContainerInGroupText(container)
					secondaryText := cl.formatSecondaryText(container)
					cl.list.AddItem(mainText, secondaryText, 0, nil)
				}
			}
		} else {
			container := group.Containers[0]
			containerItem := &ContainerItem{
				Container: container,
				IsGroup:   false,
				Level:     0,
			}
			cl.displayItems = append(cl.displayItems, containerItem)

			mainText := cl.formatMainText(container)
			secondaryText := cl.formatSecondaryText(container)
			cl.list.AddItem(mainText, secondaryText, 0, nil)
		}
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

	return fmt.Sprintf(" Containers (%d total) ", len(cl.containers))
}

func (cl *ContainerList) formatGroupText(group *ContainerGroup) string {
	icon := "📁"
	if group.IsExpanded {
		icon = "📂"
	}

	runningCount := 0
	for _, container := range group.Containers {
		if container.Status == models.StatusRunning {
			runningCount++
		}
	}

	return fmt.Sprintf("%s [yellow]%s[white] (%d containers, %d running)",
		icon, group.Name, len(group.Containers), runningCount)
}

func (cl *ContainerList) formatGroupSecondary(group *ContainerGroup) string {
	if group.IsExpanded {
		return "[dim]Click or press Enter to collapse[white]"
	}
	return "[dim]Click or press Enter to expand[white]"
}

func (cl *ContainerList) formatMainText(container *models.Container) string {
	statusColor := container.Status.Color()
	statusIcon := container.Status.Icon()

	return fmt.Sprintf("[%s]%s %s[white] (%s)",
		statusColor,
		statusIcon,
		container.ShortName(),
		container.ShortID())
}

func (cl *ContainerList) formatContainerInGroupText(container *models.Container) string {
	statusColor := container.Status.Color()
	statusIcon := container.Status.Icon()

	return fmt.Sprintf("  [%s]%s %s[white] (%s)",
		statusColor,
		statusIcon,
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
			container.FormatAge())
	}
}

func (cl *ContainerList) SetSelectedFunc(fn func(*models.Container)) {
	cl.onSelected = fn
}

func (cl *ContainerList) UpdateContainers(containers []*models.Container) {
	cl.containers = containers
	cl.buildGroups()
	cl.refreshView()
}

func (cl *ContainerList) GetSelectedContainer() *models.Container {
	index := cl.list.GetCurrentItem()
	if index >= 0 && index < len(cl.displayItems) {
		item := cl.displayItems[index]
		if !item.IsGroup {
			return item.Container
		}
	}
	return nil
}

func (cl *ContainerList) SelectContainer(id string) {
	for i, item := range cl.displayItems {
		if !item.IsGroup && item.Container != nil && item.Container.ID == id {
			cl.list.SetCurrentItem(i)
			if item.Group != nil && len(item.Group.Containers) > 1 {
				item.Group.IsExpanded = true
				cl.refreshView()
				for j, refreshedItem := range cl.displayItems {
					if !refreshedItem.IsGroup && refreshedItem.Container != nil && refreshedItem.Container.ID == id {
						cl.list.SetCurrentItem(j)
						break
					}
				}
			}
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
