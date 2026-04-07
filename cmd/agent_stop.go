package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kiev/kernus/internal/agentstop"
	"github.com/spf13/cobra"
)

var agentStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Kernus agent processes on this machine",
	Long: `Finds processes whose command line is a running 'kernus agent start' and sends SIGTERM,
waits for them to exit, then optionally force-kills stragglers.

Does not stop systemd-managed services by name; it only matches this CLI pattern.
If the agent was started with a different wrapper, use your process manager or kill manually.`,
	RunE: runAgentStop,
}

func runAgentStop(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")
	waitSec, _ := cmd.Flags().GetInt("wait")
	if waitSec < 0 {
		waitSec = 0
	}
	wait := time.Duration(waitSec) * time.Second

	pids, err := agentstop.FindRunningAgentStartPIDs(os.Getpid())
	if err != nil {
		return fmt.Errorf("find agent processes: %w", err)
	}
	if len(pids) == 0 {
		fmt.Println("No running kernus agent start process found.")
		return nil
	}
	for _, pid := range pids {
		fmt.Printf("→ Stopping PID %d\n", pid)
	}
	if err := agentstop.StopPIDs(pids, wait, force); err != nil {
		return err
	}
	fmt.Println("✓ Kernus agent stopped.")
	return nil
}

func init() {
	agentStopCmd.Flags().Bool("force", false, "After wait, SIGKILL any agent process still running")
	agentStopCmd.Flags().Int("wait", 10, "Seconds to wait after SIGTERM before failing (or before --force)")
	agentCmd.AddCommand(agentStopCmd)
}
