package details

import (
	"fmt"
	"strings"

	"github.com/kiev/kernus/internal/models"
)

func RenderOverview(b *strings.Builder, c *models.Container) {
	b.WriteString("\n  [yellow::b]Identity[white:-:-]\n")
	fmt.Fprintf(b, "    ID       : %s\n", c.ShortID())
	fmt.Fprintf(b, "    Name     : %s\n", c.ShortName())
	fmt.Fprintf(b, "    Image    : %s\n", c.ImageName())
	fmt.Fprintf(b, "    Tag      : %s\n", c.ImageTag())

	b.WriteString("\n  [yellow::b]Status[white:-:-]\n")
	statusColor := c.Status.Color()
	statusIcon := c.Status.Icon()
	fmt.Fprintf(b, "    Status   : [%s]%s %s[white]\n", statusColor, statusIcon, c.Status)
	fmt.Fprintf(b, "    State    : %s\n", c.State)
	healthColor := c.Health.Status.Color()
	healthIcon := c.Health.Status.Icon()
	fmt.Fprintf(b, "    Health   : [%s]%s %s[white]\n", healthColor, healthIcon, c.Health.Status)
	if c.ExitCode != 0 {
		fmt.Fprintf(b, "    Exit Code: [red]%d[white]\n", c.ExitCode)
	}

	b.WriteString("\n  [yellow::b]Timing[white:-:-]\n")
	if !c.Created.IsZero() {
		fmt.Fprintf(b, "    Created  : %s\n", c.Created.Format("2006-01-02 15:04:05"))
	}
	if !c.Started.IsZero() {
		fmt.Fprintf(b, "    Started  : %s\n", c.Started.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(b, "    Age      : %s\n", c.FormatAge())
	fmt.Fprintf(b, "    Uptime   : %s\n", c.FormatUptime())

	b.WriteString("\n  [yellow::b]Configuration[white:-:-]\n")
	cmd := c.Command
	if len(cmd) > 60 {
		cmd = cmd[:57] + "..."
	}
	fmt.Fprintf(b, "    Command  : %s\n", cmd)
	fmt.Fprintf(b, "    Restart  : %s", c.RestartPolicy.Name)
	if c.RestartPolicy.MaximumRetryCount > 0 {
		fmt.Fprintf(b, " (max: %d)", c.RestartPolicy.MaximumRetryCount)
	}
	b.WriteString("\n")
	if c.Stats != nil {
		fmt.Fprintf(b, "    PIDs     : %d\n", c.Stats.PIDs)
	}

	b.WriteString("\n  [yellow::b]Quick Stats[white:-:-]\n")
	if c.Stats != nil {
		fmt.Fprintf(b, "    CPU      : %.1f%%\n", c.Stats.CPU.Usage)
		fmt.Fprintf(b, "    Memory   : %s\n", c.Stats.Memory.String())
		fmt.Fprintf(b, "    Network  : %s\n", c.Stats.Network.String())
		fmt.Fprintf(b, "    PIDs     : %d\n", c.Stats.PIDs)
	} else {
		b.WriteString("    [gray]Stats not available[white]\n")
	}

	if len(c.Labels) > 0 {
		b.WriteString("\n  [yellow::b]Labels[white:-:-]\n")
		count := 0
		for k, v := range c.Labels {
			if count >= 10 {
				fmt.Fprintf(b, "    ... and %d more\n", len(c.Labels)-10)
				break
			}
			fmt.Fprintf(b, "    %s: %s\n", k, v)
			count++
		}
	}
}
