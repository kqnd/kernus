package details

import (
	"fmt"
	"strings"

	"github.com/kiev/kernus/internal/models"
)

func RenderNetwork(b *strings.Builder, c *models.Container) {
	b.WriteString("\n  [white::b]Port Mappings[white:-:-]\n\n")

	if len(c.Ports) > 0 {
		tb := NewTableBuilder("Private", "Public", "Type", "IP")
		for _, p := range c.Ports {
			pub := ""
			if p.PublicPort > 0 {
				pub = fmt.Sprintf("%d", p.PublicPort)
			}
			ip := p.IP
			if ip == "" {
				ip = "-"
			}
			tb.AddRow(fmt.Sprintf("%d", p.PrivatePort), pub, p.Type, ip)
		}
		tb.Render(b)
	} else {
		b.WriteString("  [gray]No port mappings[white]\n")
	}

	b.WriteString("\n  [white::b]Networks[white:-:-]\n\n")

	if len(c.Networks) > 0 {
		tb := NewTableBuilder("Network Name", "IP Address", "Gateway", "MAC")
		for _, n := range c.Networks {
			gw := n.Gateway
			if gw == "" {
				gw = "-"
			}
			tb.AddRow(n.Name, n.IP, gw, n.MAC)
		}
		tb.Render(b)
	} else {
		b.WriteString("  [gray]No networks[white]\n")
	}

	b.WriteString("\n  [white::b]Statistics[white:-:-]\n\n")

	if c.Stats != nil {
		net := c.Stats.Network
		tb := NewTableBuilder("Direction", "Bytes", "Packets", "Errors", "Drops")
		tb.AddRow("RX", FormatBytesInt64(net.RxBytes), FormatNumber(uint64(net.RxPackets)), FormatNumber(uint64(net.RxErrors)), FormatNumber(uint64(net.RxDropped)))
		tb.AddRow("TX", FormatBytesInt64(net.TxBytes), FormatNumber(uint64(net.TxPackets)), FormatNumber(uint64(net.TxErrors)), FormatNumber(uint64(net.TxDropped)))
		tb.Render(b)
	} else {
		b.WriteString("  [gray]Network stats not available[white]\n")
	}
}
