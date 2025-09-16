package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kern/internal/docker"
	"github.com/kern/internal/models"
	"github.com/kern/internal/tui/components"
	"github.com/rivo/tview"
	nundb "github.com/viewfromaside/nun-db-go"
)

type Config struct {
	Server        string
	Group         string
	RefreshRate   time.Duration
	MaxLogEntries int
	DockerHost    string
}

type App struct {
	tviewApp *tview.Application
	config   *Config
	nundb    *nundb.Client
	docker   *docker.Client

	header     *components.Header
	containers *components.ContainerList
	details    *components.Details

	stopChan chan struct{}
	mainGrid *tview.Grid

	isRunning     bool
	refreshTicker *time.Ticker
	focusIndex    int
	focusables    []tview.Primitive
}

func NewApp(config *Config) *App {
	if config.RefreshRate == 0 {
		config.RefreshRate = 1 * time.Second
	}
	if config.MaxLogEntries == 0 {
		config.MaxLogEntries = 1000
	}

	app := &App{
		tviewApp:   tview.NewApplication(),
		config:     config,
		isRunning:  false,
		focusIndex: 0,
		stopChan:   make(chan struct{}),
	}

	return app
}

func toPtrSlice(containers []models.Container) []*models.Container {
	out := make([]*models.Container, len(containers))
	for i := range containers {
		out[i] = &containers[i]
	}
	return out
}

func (a *App) initializeDocker() error {
	var err error
	a.docker, err = docker.NewClient(a.config.DockerHost)

	if err != nil {
		return fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	if err := a.docker.Ping(); err != nil {
		a.docker.Close()
		return fmt.Errorf("docker daemon not responding: %w", err)
	}
	return nil
}

func (a *App) startAutoRefresh() {
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}

	a.refreshTicker = time.NewTicker(a.config.RefreshRate)

	go func() {
		defer a.refreshTicker.Stop()

		for {
			select {
			case <-a.stopChan:
				return
			case <-a.refreshTicker.C:
				a.performRefresh(false)
			}
		}
	}()
}

func (a *App) performRefresh(forceRefresh bool) {
	var selectedID string
	if currentSelected := a.containers.GetSelectedContainer(); currentSelected != nil {
		selectedID = currentSelected.ID
	}

	containers, err := a.loadContainers()
	if err != nil {
		fmt.Printf("Error loading containers: %v\n", err)
		os.Exit(1)
	}

	a.tviewApp.QueueUpdateDraw(func() {
		a.containers.UpdateContainersPreserveSelection(containers, selectedID)
		a.nundb.Set("a", "b")

		if selected := a.containers.GetSelectedContainer(); selected != nil {
			a.refreshContainerStats(selected)
		}
	})
}

func (a *App) initializeComponents() {
	a.header = components.NewHeader(a.tviewApp, a.config.Server, a.config.Group)

	containers, err := a.loadContainers()
	if err != nil {
		fmt.Printf("Error loading containers, using mock data: %v\n", err)
		os.Exit(1)
	}

	a.containers = components.NewContainerList(containers)
	a.details = components.NewDetails(a.docker)

	a.containers.SetSelectedFunc(func(c *models.Container) {
		a.details.ShowContainer(c)
		a.refreshContainerStats(c)
	})

	a.focusables = []tview.Primitive{
		a.containers.GetView(),
		a.details.GetView(),
	}
}

func (a *App) loadContainers() ([]*models.Container, error) {
	if a.docker == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	containers, err := a.docker.ListContainers(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return toPtrSlice(containers), nil
}

func (a *App) refreshContainerStats(container *models.Container) {
	if a.docker == nil || container.Status != models.StatusRunning {
		return
	}

	go func() {
		if stats, err := a.docker.GetContainerStats(container.ID); err == nil {
			container.Stats = stats
			a.tviewApp.QueueUpdateDraw(func() {
				a.details.ShowContainer(container)
			})
		}
	}()
}

func (a *App) SetNunDBClient(client *nundb.Client) {
	a.nundb = client
}

func (a *App) setupLayout() {
	a.mainGrid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(40, 0).
		SetBorders(false)

	a.mainGrid.AddItem(a.header.GetView(), 0, 0, 1, 2, 0, 0, false)
	a.mainGrid.AddItem(a.containers.GetView(), 1, 0, 1, 1, 0, 0, true)
	a.mainGrid.AddItem(a.details.GetView(), 1, 1, 1, 1, 0, 0, false)

	a.tviewApp.SetRoot(a.mainGrid, true).EnableMouse(true)
}

func (a *App) setupKeyBindings() {
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			a.quit()
			return nil
		case tcell.KeyTab:
			a.switchFocus()
			return nil
		case tcell.KeyF1, tcell.KeyF2, tcell.KeyF3, tcell.KeyF4, tcell.KeyF6:
			a.switchDetailsTab(event.Key())
			return nil
		}

		switch event.Rune() {
		case 'q', 'Q':
			a.quit()
			return nil
		case '1', '2', '3', '4', '5':
			tabIndex := int(event.Rune() - '1')
			a.details.SwitchTab(tabIndex)
			return nil
		case 's', 'S':
			a.handleContainerAction("start")
			return nil
		case 't', 'T':
			a.handleContainerAction("stop")
			return nil
		case 'p', 'P':
			a.handleContainerAction("pause")
			return nil
		case 'u', 'U':
			a.handleContainerAction("unpause")
			return nil
		case 'd', 'D':
			a.handleContainerAction("remove")
			return nil
		}

		return event
	})
}

func (a *App) switchDetailsTab(key tcell.Key) {
	var tabIndex int
	switch key {
	case tcell.KeyF1:
		tabIndex = 0
	case tcell.KeyF2:
		tabIndex = 1
	case tcell.KeyF3:
		tabIndex = 2
	case tcell.KeyF4:
		tabIndex = 3
	case tcell.KeyF6:
		tabIndex = 4
	default:
		return
	}
	a.details.SwitchTab(tabIndex)
}

func (a *App) handleContainerAction(action string) {
	if a.docker == nil {
		return
	}

	selected := a.containers.GetSelectedContainer()
	if selected == nil {
		return
	}

	go func() {
		var err error
		switch action {
		case "start":
			if selected.Status != models.StatusRunning {
				err = a.docker.StartContainer(selected.ID)
			}
		case "stop":
			if selected.Status == models.StatusRunning {
				err = a.docker.StopContainer(selected.ID)
			}
		case "pause":
			if selected.Status == models.StatusRunning {
				err = a.docker.PauseContainer(selected.ID)
			}
		case "unpause":
			if selected.Status == models.StatusPaused {
				err = a.docker.UnpauseContainer(selected.ID)
			}
		case "remove":
			if selected.Status != models.StatusRunning {
				err = a.docker.RemoveContainer(selected.ID, false)
			}
		}

		if err != nil {
			fmt.Printf("Error performing %s action: %v\n", action, err)
		} else {
			time.Sleep(500 * time.Millisecond)
			a.forceRefresh()
		}
	}()
}

func (a *App) forceRefresh() {
	a.performRefresh(true)
}

func (a *App) switchFocus() {
	a.focusIndex = (a.focusIndex + 1) % len(a.focusables)
	a.tviewApp.SetFocus(a.focusables[a.focusIndex])
}

func (a *App) quit() {
	a.isRunning = false

	if a.stopChan != nil {
		select {
		case <-a.stopChan:
		default:
			close(a.stopChan)
		}
	}

	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}

	if a.header != nil {
		a.header.Stop()
	}

	if a.tviewApp != nil {
		a.tviewApp.Stop()
	}
}

func (a *App) Run() error {
	if err := a.initializeDocker(); err != nil {
		return fmt.Errorf("docker initialization failed: %w", err)
	}

	defer func() {
		if a.docker != nil {
			a.docker.Close()
		}
	}()

	a.initializeComponents()
	a.setupLayout()
	a.setupKeyBindings()

	a.tviewApp.SetFocus(a.focusables[0])

	a.isRunning = true
	a.startAutoRefresh()

	if err := a.tviewApp.Run(); err != nil {
		return err
	}
	return nil
}
