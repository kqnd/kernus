package tui

import (
	"time"

	"github.com/kern/internal/tui/components"
	"github.com/rivo/tview"
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

	header   *components.Header
	machines *components.MachineList

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

func (a *App) initializeComponents() {
	a.header = components.NewHeader(a.tviewApp, a.config.Server, a.config.Group)
	a.machines = components.NewMachineList()
}

func (a *App) setupLayout() {
	a.mainGrid = tview.NewGrid().
		SetRows(3, 0, 8).
		SetColumns(30, 0).
		SetBorders(false)

	a.mainGrid.AddItem(a.header.GetView(), 0, 0, 1, 2, 0, 0, false)
	a.mainGrid.AddItem(a.machines.GetView(), 1, 0, 1, 1, 0, 0, true)
	a.tviewApp.SetRoot(a.mainGrid, true).EnableMouse(true)
}

func (a *App) quit() {
	a.isRunning = false

	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}
	a.header.Stop()
	a.tviewApp.Stop()
}

func (a *App) Run() error {
	a.initializeComponents()
	a.setupLayout()

	a.isRunning = true
	if err := a.tviewApp.Run(); err != nil {
		return err
	}
	return nil
}
