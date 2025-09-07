package components

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

type Header struct {
	server string
	group  string
	view   *tview.TextView
	ticker *time.Ticker
	stopCh chan bool
}

func NewHeader(server, group string) *Header {
	h := &Header{
		view:   tview.NewTextView(),
		server: server,
		group:  group,
		stopCh: make(chan bool),
	}

	h.setupView()
	h.startClock()

	return h
}

func (h *Header) setupView() {
	h.view.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetBorder(true).
		SetTitle("Connection Info")

	h.updateContent()
}

func (h *Header) updateContent() {
	currentTime := time.Now().Format("15:04:05")
	headerText := fmt.Sprintf("[yellow]Server:[white] %s", h.server)
	if h.group != "" {
		headerText += fmt.Sprintf(" [yellow]| Group:[white] %s", h.group)
	}

	headerText += fmt.Sprintf(" [yellow]| Time:[white] %s", currentTime)
	headerText += " [yellow]| Status:[green] Connected[white]"

	h.view.SetText(headerText)
}

func (h *Header) startClock() {
	h.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-h.ticker.C:
				h.updateContent()
			case <-h.stopCh:
				return
			}
		}
	}()
}

func (h *Header) Stop() {
	if h.ticker != nil {
		h.ticker.Stop()
	}
	close(h.stopCh)
}

func (h *Header) GetView() tview.Primitive {
	return h.view
}
