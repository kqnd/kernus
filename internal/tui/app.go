package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
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
}

type App struct {
	tviewApp *tview.Application
	config   *Config
	nundb    *nundb.Client

	header     *components.Header
	containers *components.ContainerList
	details    *components.Details

	mainGrid *tview.Grid

	isRunning     bool
	refreshTicker *time.Ticker
	focusIndex    int
	focusables    []tview.Primitive
}

func NewApp(config *Config) *App {

	if config.RefreshRate == 0 {
		config.RefreshRate = 30 * time.Second
	}
	if config.MaxLogEntries == 0 {
		config.MaxLogEntries = 1000
	}

	app := &App{
		tviewApp:   tview.NewApplication(),
		config:     config,
		isRunning:  false,
		focusIndex: 0,
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

func (a *App) initializeComponents() {
	a.header = components.NewHeader(a.tviewApp, a.config.Server, a.config.Group)
	a.containers = components.NewContainerList(toPtrSlice(models.MockContainers()))
	a.details = components.NewDetails()

	a.containers.SetSelectedFunc(func(c *models.Container) {
		a.details.ShowContainer(c)
	})

	a.focusables = []tview.Primitive{
		a.containers.GetView(),
		a.details.GetView(),
	}
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
		case tcell.KeyF5, tcell.KeyCtrlR:
			a.refreshData()
			return nil
		}

		switch event.Rune() {
		case 'q', 'Q':
			a.quit()
			return nil
		case 'r', 'R':
			a.refreshData()
			return nil
		}

		return event
	})
}

func (a *App) switchFocus() {
	a.focusIndex = (a.focusIndex + 1) % len(a.focusables)
	a.tviewApp.SetFocus(a.focusables[a.focusIndex])
}

func (a *App) refreshData() {
	containers := toPtrSlice(models.MockContainers())
	a.containers.UpdateContainers(containers)
	a.tviewApp.Draw()
}

func (a *App) quit() {
	a.isRunning = false

	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}
	a.header.Stop()
	a.tviewApp.Stop()
}

func (a *App) startAutoRefresh() {
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}

	a.refreshTicker = time.NewTicker(a.config.RefreshRate)
	go func() {
		for range a.refreshTicker.C {
			if !a.isRunning {
				return
			}
			a.refreshData()
		}
	}()
}

func (a *App) Run() error {
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
