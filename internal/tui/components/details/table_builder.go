package details

import (
	"fmt"
	"strings"

	"github.com/kern/internal/models"
)

type TableBuilder struct {
	formatter *Formatter
}

func NewTableBuilder() *TableBuilder {
	return &TableBuilder{
		formatter: NewFormatter(),
	}
}

func (t *TableBuilder) BuildPortsTable(ports []models.Port) string {
	if len(ports) == 0 {
		return "  [gray]No ports exposed[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Private    Public     Type    IP[white]\n")
	result.WriteString("  [gray]─────────────────────────────────────[white]\n")

	for _, port := range ports {
		publicStr := "─"
		ipStr := "─"

		if port.PublicPort > 0 {
			publicStr = fmt.Sprintf("%d", port.PublicPort)
		}
		if port.IP != "" {
			ipStr = port.IP
		}

		result.WriteString(fmt.Sprintf("  %-9d  %-9s  %-6s  %s\n",
			port.PrivatePort, publicStr, port.Type, ipStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (t *TableBuilder) BuildNetworksTable(networks []models.Network) string {
	if len(networks) == 0 {
		return "  [gray]No networks configured[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Network          IP Address       Gateway[white]\n")
	result.WriteString("  [gray]───────────────────────────────────────────────[white]\n")

	for _, network := range networks {
		ipStr := network.IPAddress
		gatewayStr := network.Gateway

		if ipStr == "" {
			ipStr = "─"
		}
		if gatewayStr == "" {
			gatewayStr = "─"
		}

		result.WriteString(fmt.Sprintf("  %-15s  %-15s  %s\n",
			t.formatter.TruncateString(network.Name, 15), ipStr, gatewayStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

func (t *TableBuilder) BuildMountsTable(mounts []models.Mount) string {
	if len(mounts) == 0 {
		return "  [gray]No mounts configured[white]"
	}

	var result strings.Builder
	result.WriteString("  [gray]Source                    Destination              Type    Mode[white]\n")
	result.WriteString("  [gray]─────────────────────────────────────────────────────────────────────[white]\n")

	for _, mount := range mounts {
		modeStr := mount.Mode
		if modeStr == "" {
			if mount.RW {
				modeStr = "rw"
			} else {
				modeStr = "ro"
			}
		}

		result.WriteString(fmt.Sprintf("  %-24s  %-23s  %-6s  %s\n",
			t.formatter.TruncateString(mount.Source, 24),
			t.formatter.TruncateString(mount.Destination, 23),
			mount.Type,
			modeStr))
	}

	return strings.TrimSuffix(result.String(), "\n")
}
