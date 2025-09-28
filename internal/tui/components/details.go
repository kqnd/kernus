package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kqnd/kernus/internal/docker"
	"github.com/kqnd/kernus/internal/models"
	"github.com/kqnd/kernus/internal/tui/components/details"
	"github.com/rivo/tview"
)

type Details struct {
	view             *tview.TextView
	currentContainer *models.Container
	docker           *docker.Client
	tabs             []string
	currentTab       int

	overviewTab *details.OverviewTab
	statsTab    *details.StatsTab
	networkTab  *details.NetworkTab
	storageTab  *details.StorageTab
	logsTab     *details.LogsTab
}

const (
	TAB_OVERVIEW = iota
	TAB_STATS
	TAB_NETWORK
	TAB_STORAGE
	TAB_LOGS
)

func NewDetails(docker *docker.Client) *Details {
	d := &Details{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetWordWrap(true).
			SetScrollable(true).
			SetChangedFunc(func() {
			}),
		tabs:       []string{"Overview", "Stats", "Network", "Storage", "Logs"},
		currentTab: TAB_OVERVIEW,
		docker:     docker,

		overviewTab: details.NewOverviewTab(),
		statsTab:    details.NewStatsTab(),
		networkTab:  details.NewNetworkTab(),
		storageTab:  details.NewStorageTab(),
		logsTab:     details.NewLogsTab(),
	}

	d.view.SetBorder(true).SetTitle(" Container Details ")
	d.view.SetText(d.buildEmptyState())

	d.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			d.PrevTab()
			return nil
		case tcell.KeyRight:
			d.NextTab()
			return nil
		}

		switch event.Rune() {
		case 'r', 'R':
			if d.currentContainer != nil {
				go func() {
					if d.currentTab == TAB_LOGS {
						d.refreshLogs()
					} else {
						d.restartDocker()
					}
				}()
			}
			return nil
		}
		return event
	})

	return d
}

func (d *Details) restartDocker() {
	if d.currentContainer == nil {
		return
	}

	d.currentContainer.Status = models.StatusRestarting
	d.docker.RestartContainer(d.currentContainer.ID)
	d.updateView()
}

func (d *Details) refreshLogs() {
	if d.currentContainer == nil {
		return
	}

	if logs, err := d.docker.RefreshContainerLogs(d.currentContainer.ID, 100); err == nil {
		d.currentContainer.Logs = logs
		d.updateView()
	}
}

func (d *Details) ShowContainer(container *models.Container) {
	d.currentContainer = container
	d.updateView()
}

func (d *Details) SwitchTab(tab int) {
	if tab >= 0 && tab < len(d.tabs) {
		d.currentTab = tab
		d.updateView()
	}
}

func (d *Details) NextTab() {
	d.currentTab = (d.currentTab + 1) % len(d.tabs)
	d.updateView()
}

func (d *Details) PrevTab() {
	d.currentTab = (d.currentTab - 1 + len(d.tabs)) % len(d.tabs)
	d.updateView()
}

func (d *Details) updateView() {
	if d.currentContainer == nil {
		d.view.SetTitle(" Container Details ")
		d.view.SetText(d.buildEmptyState())
		return
	}

	title := fmt.Sprintf(" %s - %s ", d.tabs[d.currentTab], d.currentContainer.ShortName())
	d.view.SetTitle(title)

	var content string
	switch d.currentTab {
	case TAB_OVERVIEW:
		content = d.buildTabHeader() + d.overviewTab.Render(d.currentContainer)
	case TAB_STATS:
		content = d.buildTabHeader() + d.statsTab.Render(d.currentContainer)
	case TAB_NETWORK:
		content = d.buildTabHeader() + d.networkTab.Render(d.currentContainer)
	case TAB_STORAGE:
		content = d.buildTabHeader() + d.storageTab.Render(d.currentContainer)
	case TAB_LOGS:
		content = d.buildTabHeader() + d.logsTab.Render(d.currentContainer)
	default:
		content = d.buildTabHeader() + d.overviewTab.Render(d.currentContainer)
	}

	d.view.SetText(content)
}

func (d *Details) buildEmptyState() string {
	return `[yellow]Container Details[white]

[gray]┌─────────────────────────────────────┐[white]
[gray]│[white]  No container selected              [gray]│[white]
[gray]│[white]                                     [gray]│[white]
[gray]│[white]  Select a container from the list   [gray]│[white]
[gray]│[white]  to view detailed information       [gray]│[white]
[gray]│[white]                                     [gray]│[white]
[gray]│[white]  Available tabs:                    [gray]│[white]
[gray]│[white]  • Overview - Basic info & status   [gray]│[white]
[gray]│[white]  • Stats    - Resource usage        [gray]│[white]
[gray]│[white]  • Network  - Network configuration [gray]│[white]
[gray]│[white]  • Storage  - Mounts & volumes      [gray]│[white]
[gray]└─────────────────────────────────────┘[white]

[darkgray]Use [white]Left/Right arrows[darkgray] to switch between tabs[white]`
}

func (d *Details) buildTabHeader() string {
	var tabs []string
	for i, tab := range d.tabs {
		if i == d.currentTab {
			tabs = append(tabs, fmt.Sprintf("[white]> %s <[white]", tab))
		} else {
			tabs = append(tabs, fmt.Sprintf("[gray]  %s  [white]", tab))
		}
	}
	return fmt.Sprintf("%s\n\n", strings.Join(tabs, ""))
}

func (d *Details) GetView() tview.Primitive {
	return d.view
}
