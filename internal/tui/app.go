package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kiev/kernus/internal/docker"
	"github.com/kiev/kernus/internal/models"
	"github.com/kiev/kernus/internal/tui/components"
	"github.com/rivo/tview"
)

type Config struct {
	Group        string
	RefreshRate  int
	MaxLogLines  int
	DockerHost   string
	UseMock      bool
	ShowMachines bool
	Session      *models.Session
}

type App struct {
	config        Config
	tviewApp      *tview.Application
	docker        docker.DockerClient
	header        *components.Header
	statusBar     *components.StatusBar
	containerList *components.ContainerList
	machineList   *components.MachineList
	details       *components.Details
	modal         *components.Modal
	grid          *tview.Grid
	mainContent   *tview.Flex
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	containers    []models.Container
	machines      []models.Machine
	connected     bool
	focusLeft     bool
}

func Run(cfg Config) error {
	if cfg.RefreshRate <= 0 {
		cfg.RefreshRate = 3
	}
	if cfg.MaxLogLines <= 0 {
		cfg.MaxLogLines = 1000
	}

	app := &App{
		config:    cfg,
		tviewApp:  tview.NewApplication(),
		connected: false,
		focusLeft: true,
	}

	return app.run()
}

func (a *App) run() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer a.cancel()

	err := a.initializeDocker(a.ctx)
	if err != nil {
		a.config.UseMock = true
		err = a.initializeDocker(a.ctx)
		if err != nil {
			return fmt.Errorf("cannot initialize: %w", err)
		}
	}
	defer a.docker.Close()

	a.initializeComponents()
	a.setupLayout()
	a.setupKeyBindings()
	a.startAutoRefresh(a.ctx)

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.performRefresh(a.ctx)
	}()

	return a.tviewApp.SetRoot(a.grid, true).EnableMouse(false).Run()
}

func (a *App) initializeDocker(ctx context.Context) error {
	if a.config.UseMock {
		a.docker = docker.NewMockClient()
		a.connected = true
		return nil
	}

	client, err := docker.NewClient(a.config.DockerHost)
	if err != nil {
		return err
	}
	a.docker = client
	a.connected = true
	return nil
}

func (a *App) initializeComponents() {
	username := ""
	role := ""
	if a.config.Session != nil {
		username = a.config.Session.Username
	}

	a.header = components.NewHeader(username, role, a.tviewApp)
	a.header.SetConnected(a.connected)
	if a.config.Session != nil {
		remaining := time.Until(a.config.Session.ExpiresAt)
		if remaining < time.Hour && remaining > 0 {
			a.header.SetSessionWarning(a.config.Session.FormatExpiry())
		}
	}

	a.statusBar = components.NewStatusBar(a.config.ShowMachines)

	if a.config.ShowMachines {
		a.machineList = components.NewMachineList()
		a.machineList.SetSelectedFunc(func(m *models.Machine) {
			a.details.UpdateMachine(m)
		})
	} else {
		a.containerList = components.NewContainerList()
		a.containerList.SetSelectedFunc(func(c *models.Container) {
			a.details.UpdateContainer(c)
		})
	}

	a.details = components.NewDetails(a.config.ShowMachines)
	a.modal = components.NewModal()
}

func (a *App) setupLayout() {
	a.grid = tview.NewGrid().
		SetRows(3, 0, 1).
		SetColumns(42, 0).
		SetBorders(true)

	a.grid.AddItem(a.header.View, 0, 0, 1, 2, 0, 0, false)

	if a.config.ShowMachines {
		a.grid.AddItem(a.machineList.View, 1, 0, 1, 1, 0, 0, true)
	} else {
		a.grid.AddItem(a.containerList.View, 1, 0, 1, 1, 0, 0, true)
	}

	a.grid.AddItem(a.details.View, 1, 1, 1, 1, 0, 0, false)
	a.grid.AddItem(a.statusBar.View, 2, 0, 1, 2, 0, 0, false)
}

func (a *App) setupKeyBindings() {
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.modal.IsVisible() {
			return event
		}

		switch event.Key() {
		case tcell.KeyEscape:
			a.quit()
			return nil
		case tcell.KeyTab:
			a.toggleFocus()
			return nil
		case tcell.KeyF1:
			a.details.SetTab(0)
			return nil
		case tcell.KeyF2:
			a.details.SetTab(1)
			return nil
		case tcell.KeyF3:
			a.details.SetTab(2)
			return nil
		case tcell.KeyF4:
			a.details.SetTab(3)
			return nil
		case tcell.KeyF5:
			a.details.SetTab(4)
			return nil
		}

		switch event.Rune() {
		case 'q':
			a.quit()
			return nil
		case '1':
			a.details.SetTab(0)
			return nil
		case '2':
			a.details.SetTab(1)
			return nil
		case '3':
			a.details.SetTab(2)
			return nil
		case '4':
			a.details.SetTab(3)
			return nil
		case '5':
			a.details.SetTab(4)
			return nil
		case 's':
			a.containerAction(func(ctx context.Context, id string) error {
				return a.docker.StartContainer(ctx, id)
			})
			return nil
		case 't':
			a.containerActionWithConfirm("Stop", func(ctx context.Context, id string) error {
				return a.docker.StopContainer(ctx, id)
			})
			return nil
		case 'p':
			a.containerAction(func(ctx context.Context, id string) error {
				return a.docker.PauseContainer(ctx, id)
			})
			return nil
		case 'u':
			a.containerAction(func(ctx context.Context, id string) error {
				return a.docker.UnpauseContainer(ctx, id)
			})
			return nil
		case 'd':
			a.containerActionWithConfirm("Remove", func(ctx context.Context, id string) error {
				return a.docker.RemoveContainer(ctx, id, false)
			})
			return nil
		case 'r':
			go a.performRefresh(a.ctx)
			return nil
		case 'i':
			a.showProfile()
			return nil
		case '?':
			a.showHelp()
			return nil
		case 'o':
			a.showOnboarding()
			return nil
		}

		return event
	})
}

func (a *App) toggleFocus() {
	a.focusLeft = !a.focusLeft
	if a.focusLeft {
		if a.config.ShowMachines {
			a.tviewApp.SetFocus(a.machineList.View)
		} else {
			a.tviewApp.SetFocus(a.containerList.View)
		}
	} else {
		a.tviewApp.SetFocus(a.details.View)
	}
}

func (a *App) containerAction(action func(ctx context.Context, id string) error) {
	if a.config.ShowMachines || a.containerList == nil {
		return
	}
	c := a.containerList.GetSelectedContainer()
	if c == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		actionErr := action(ctx, c.ID)
		if actionErr != nil {
			a.tviewApp.QueueUpdateDraw(func() {
				a.statusBar.SetError(actionErr.Error())
			})
			return
		}
		a.performRefresh(a.ctx)
	}()
}

func (a *App) containerActionWithConfirm(actionName string, action func(ctx context.Context, id string) error) {
	if a.config.ShowMachines || a.containerList == nil {
		return
	}
	c := a.containerList.GetSelectedContainer()
	if c == nil {
		return
	}

	a.modal.Show(a.tviewApp, a.grid, fmt.Sprintf("%s container \"%s\"?", actionName, c.ShortName()),
		func() {
			go func() {
				ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
				defer cancel()
				actionErr := action(ctx, c.ID)
				if actionErr != nil {
					a.tviewApp.QueueUpdateDraw(func() {
						a.statusBar.SetError(actionErr.Error())
					})
					return
				}
				a.performRefresh(a.ctx)
			}()
		},
		func() {})
}

func (a *App) showProfile() {
	if a.config.Session == nil {
		return
	}

	user := models.MockUser()
	user.Username = a.config.Session.Username

	panel := components.NewProfilePanel(&user, a.config.Session)
	panel.OnClose = func() {
		a.tviewApp.SetRoot(a.grid, true)
	}
	panel.OnLogout = func() {
		a.tviewApp.QueueUpdateDraw(func() {
			a.quit()
		})
	}

	pages := tview.NewPages().
		AddPage("main", a.grid, true, true).
		AddPage("profile", panel.Modal, true, true)

	a.tviewApp.SetRoot(pages, true)
}

func (a *App) showHelp() {
	helpText := `[yellow]Keyboard Shortcuts[white]

[yellow]Navigation[white]
  Tab        Toggle focus between panels
  1-5 / F1-F5  Switch detail tabs
  ↑/↓        Navigate list
  ←/→        Collapse/Expand groups

[yellow]Container Actions[white]
  s          Start container
  t          Stop container
  p          Pause container
  u          Unpause container
  d          Remove container

[yellow]General[white]
  r          Force refresh
  i          User profile
  ?          This help
  q / Esc    Quit`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.tviewApp.SetRoot(a.grid, true)
		})

	pages := tview.NewPages().
		AddPage("main", a.grid, true, true).
		AddPage("help", modal, true, true)

	a.tviewApp.SetRoot(pages, true)
}

func (a *App) startAutoRefresh(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.RefreshRate) * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.performRefresh(ctx)
			}
		}
	}()
}

func (a *App) performRefresh(ctx context.Context) {
	if a.config.ShowMachines {
		a.refreshMachines()
		return
	}

	a.mu.Lock()
	var selectedID string
	if a.containerList != nil {
		c := a.containerList.GetSelectedContainer()
		if c != nil {
			selectedID = c.ID
		}
	}
	a.mu.Unlock()

	containers, err := a.docker.ListContainers(ctx, false)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			a.header.SetConnected(false)
		})
		return
	}

	a.mu.Lock()
	a.containers = containers
	a.mu.Unlock()

	a.tviewApp.QueueUpdateDraw(func() {
		a.header.SetConnected(true)
		ptrs := make([]*models.Container, len(containers))
		for i := range containers {
			ptrs[i] = &containers[i]
		}
		a.containerList.UpdateContainersPreserveSelection(ptrs, selectedID)

		if len(containers) == 0 {
			a.details.SetOnboardingMessage(`  [yellow::b]Nenhum servidor conectado ainda[white:-:-]

  Para comecar a monitorar seus containers:

  1. [white::b]Crie um Agent Token:[-:-:-]
     kernus token create "prod-server-01"

  2. [white::b]Configure o token neste host:[-:-:-]
     kernus token kn_live_xxx --server https://api.kernus.app --host prod-server-01

  3. [white::b]Inicie o agente:[-:-:-]
     kernus agent start

  [gray]Dica: pressione 'o' para ver esse guia novamente.[white]`)
		} else {
			a.details.SetOnboardingMessage("")
		}

		selected := a.containerList.GetSelectedContainer()
		if selected != nil {
			a.details.UpdateContainer(selected)
		}
	})
}

func (a *App) showOnboarding() {
	onboarding := `[yellow]Onboarding Kernus[white]

1) Gere um token da organizacao:
   kernus token create "prod-server-01"

2) Configure no servidor alvo:
   kernus token kn_live_xxx --server https://api.kernus.io --host prod-server-01

3) Inicie o agente:
   kernus agent start

Assim que as metricas chegarem, o painel sai do estado vazio automaticamente.`

	modal := tview.NewModal().
		SetText(onboarding).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.tviewApp.SetRoot(a.grid, true)
		})

	pages := tview.NewPages().
		AddPage("main", a.grid, true, true).
		AddPage("onboarding", modal, true, true)

	a.tviewApp.SetRoot(pages, true)
}

func (a *App) refreshMachines() {
	machines := models.MockMachines()

	if a.config.Group != "" {
		var filtered []models.Machine
		for _, m := range machines {
			if m.Group == a.config.Group {
				filtered = append(filtered, m)
			}
		}
		machines = filtered
	}

	a.mu.Lock()
	a.machines = machines
	a.mu.Unlock()

	a.tviewApp.QueueUpdateDraw(func() {
		ptrs := make([]*models.Machine, len(machines))
		for i := range machines {
			ptrs[i] = &machines[i]
		}
		a.machineList.UpdateMachines(ptrs)

		selected := a.machineList.GetSelectedMachine()
		if selected != nil {
			a.details.UpdateMachine(selected)
		}
	})
}

func (a *App) quit() {
	a.cancel()
	a.header.Stop()
	a.tviewApp.Stop()
}
