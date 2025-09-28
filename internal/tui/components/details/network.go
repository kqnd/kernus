package details

import (
	"fmt"
	"strings"

	"github.com/kqnd/kernus/internal/models"
)

type NetworkTab struct {
	formatter    *Formatter
	tableBuilder *TableBuilder
}

func NewNetworkTab() *NetworkTab {
	return &NetworkTab{
		formatter:    NewFormatter(),
		tableBuilder: NewTableBuilder(),
	}
}

func (n *NetworkTab) Render(container *models.Container) string {
	if container == nil {
		return ""
	}

	sections := []string{
		n.renderPortMappings(container),
		n.renderNetworks(container),
		n.renderNetworkStats(container),
	}

	return fmt.Sprintf("[yellow]Network Configuration[white]\n\n%s", strings.Join(sections, "\n\n"))
}

func (n *NetworkTab) renderPortMappings(c *models.Container) string {
	portsTable := n.tableBuilder.BuildPortsTable(c.Ports)
	return fmt.Sprintf("[yellow]Port Mappings (%d)[white]\n%s", len(c.Ports), portsTable)
}

func (n *NetworkTab) renderNetworks(c *models.Container) string {
	networksTable := n.tableBuilder.BuildNetworksTable(c.Networks)
	return fmt.Sprintf("[yellow]Networks (%d)[white]\n%s", len(c.Networks), networksTable)
}

func (n *NetworkTab) renderNetworkStats(c *models.Container) string {
	if c.Stats == nil {
		return "[yellow]Network Statistics[white]\n  [gray]Network statistics not available[white]"
	}

	return fmt.Sprintf(`[yellow]Network Statistics[white]
  Received : %s (%s packets, %d errors)
  Sent     : %s (%s packets, %d errors)`,
		n.formatter.FormatBytes(c.Stats.Network.RxBytes),
		n.formatter.FormatNumber(c.Stats.Network.RxPackets),
		c.Stats.Network.RxErrors,
		n.formatter.FormatBytes(c.Stats.Network.TxBytes),
		n.formatter.FormatNumber(c.Stats.Network.TxPackets),
		c.Stats.Network.TxErrors)
}
