//go:build unix && !linux

package agentstop

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// FindRunningAgentStartPIDs uses pgrep -f on BSD/macOS (no /proc cmdline).
func FindRunningAgentStartPIDs(excludePID int) ([]int, error) {
	// Must not match `kernus agent stop`; require literal "agent start" sequence in argv.
	cmd := exec.Command("pgrep", "-f", `kernus agent start`)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("pgrep: %w", err)
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		if pid == excludePID {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}
