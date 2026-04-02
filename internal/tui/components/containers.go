package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kiev/kernus/internal/models"
	"github.com/rivo/tview"
)

type ContainerList struct {
	View         *tview.List
	containers   []*models.Container
	groups       map[string][]*models.Container
	groupOrder   []string
	expanded     map[string]bool
	items        []listItem
	selectedFunc func(*models.Container)
}

type listItem struct {
	isGroup   bool
	groupName string
	container *models.Container
}

func NewContainerList() *ContainerList {
	cl := &ContainerList{
		View:     tview.NewList(),
		expanded: make(map[string]bool),
	}

	cl.View.SetBorder(false)
	cl.View.SetTitle(" Containers ")
	cl.View.SetHighlightFullLine(true)
	cl.View.ShowSecondaryText(false)
	cl.View.SetSelectedBackgroundColor(tcell.ColorDarkBlue)

	cl.View.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(cl.items) {
			item := cl.items[index]
			if !item.isGroup && item.container != nil && cl.selectedFunc != nil {
				cl.selectedFunc(item.container)
			}
		}
	})

	cl.View.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(cl.items) {
			item := cl.items[index]
			if item.isGroup {
				cl.expanded[item.groupName] = !cl.expanded[item.groupName]
				cl.rebuildList()
			}
		}
	})

	cl.View.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		idx := cl.View.GetCurrentItem()
		if idx < 0 || idx >= len(cl.items) {
			return event
		}
		item := cl.items[idx]

		switch event.Key() {
		case tcell.KeyRight:
			if item.isGroup && !cl.expanded[item.groupName] {
				cl.expanded[item.groupName] = true
				cl.rebuildList()
				return nil
			}
		case tcell.KeyLeft:
			if item.isGroup && cl.expanded[item.groupName] {
				cl.expanded[item.groupName] = false
				cl.rebuildList()
				return nil
			}
			if !item.isGroup && item.container != nil {
				for gn, members := range cl.groups {
					for _, m := range members {
						if m.ID == item.container.ID {
							if len(members) > 1 {
								cl.expanded[gn] = false
								cl.rebuildList()
								for i, it := range cl.items {
									if it.isGroup && it.groupName == gn {
										cl.View.SetCurrentItem(i)
										break
									}
								}
								return nil
							}
						}
					}
				}
			}
		}
		return event
	})

	return cl
}

func (cl *ContainerList) SetSelectedFunc(fn func(*models.Container)) {
	cl.selectedFunc = fn
}

func (cl *ContainerList) GetSelectedContainer() *models.Container {
	idx := cl.View.GetCurrentItem()
	if idx < 0 || idx >= len(cl.items) {
		return nil
	}
	item := cl.items[idx]
	if item.isGroup {
		return nil
	}
	return item.container
}

func (cl *ContainerList) UpdateContainersPreserveSelection(containers []*models.Container, selectedID string) {
	cl.containers = containers
	cl.buildGroups()
	cl.rebuildList()

	if selectedID != "" {
		for i, item := range cl.items {
			if !item.isGroup && item.container != nil && item.container.ID == selectedID {
				cl.View.SetCurrentItem(i)
				return
			}
		}
	}
}

func (cl *ContainerList) buildGroups() {
	prefixMap := make(map[string][]*models.Container)

	for _, c := range cl.containers {
		prefix := extractPrefix(c.ShortName())
		prefixMap[prefix] = append(prefixMap[prefix], c)
	}

	cl.groups = make(map[string][]*models.Container)
	for prefix, members := range prefixMap {
		if len(members) > 1 {
			cl.groups[prefix] = members
		} else {
			cl.groups[members[0].ShortName()] = members
		}
	}

	cl.groupOrder = make([]string, 0, len(cl.groups))
	for g := range cl.groups {
		cl.groupOrder = append(cl.groupOrder, g)
	}
	sort.Strings(cl.groupOrder)
}

func extractPrefix(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) <= 1 {
		return name
	}
	return strings.Join(parts[:len(parts)-1], "-")
}

func (cl *ContainerList) rebuildList() {
	cl.View.Clear()
	cl.items = nil

	total := len(cl.containers)
	cl.View.SetTitle(fmt.Sprintf(" Containers (%d total) ", total))

	for _, groupName := range cl.groupOrder {
		members := cl.groups[groupName]

		if len(members) == 1 {
			c := members[0]
			icon := c.Status.Icon()
			color := c.Status.Color()
			portStr := ""
			if p := c.MainPort(); p != "" {
				portStr = "  " + p
			}
			text := fmt.Sprintf("[%s]%s %s[white]%s", color, icon, c.ShortName(), portStr)
			cl.View.AddItem(text, "", 0, nil)
			cl.items = append(cl.items, listItem{container: c})
			continue
		}

		runCount := 0
		for _, m := range members {
			if m.Status == models.StatusRunning {
				runCount++
			}
		}

		expanded := cl.expanded[groupName]
		folderIcon := "📁"
		if expanded {
			folderIcon = "📂"
		}

		groupText := fmt.Sprintf("%s [yellow]%s[white] (%d)  %d running", folderIcon, groupName, len(members), runCount)
		cl.View.AddItem(groupText, "", 0, nil)
		cl.items = append(cl.items, listItem{isGroup: true, groupName: groupName})

		if expanded {
			for _, c := range members {
				icon := c.Status.Icon()
				color := c.Status.Color()
				portStr := ""
				if p := c.MainPort(); p != "" {
					portStr = "  " + p
				}
				text := fmt.Sprintf("  [%s]%s %s[white]%s", color, icon, c.ShortName(), portStr)
				cl.View.AddItem(text, "", 0, nil)
				cl.items = append(cl.items, listItem{container: c})
			}
		}
	}
}
