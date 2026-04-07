//go:build windows

package agentstop

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FindRunningAgentStartPIDs uses PowerShell CIM; matches command lines containing
// kernus + agent + start (not stop).
func FindRunningAgentStartPIDs(excludePID int) ([]int, error) {
	ps := `$p = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
		$_.CommandLine -and
		$_.CommandLine -match 'kernus' -and
		$_.CommandLine -match ' agent start' -and
		$_.CommandLine -notmatch ' agent stop'
	}
	if ($null -eq $p) { exit 0 }
	$p | ForEach-Object { $_.ProcessId }`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell: %w", err)
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
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
