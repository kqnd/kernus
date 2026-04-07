//go:build linux

package agentstop

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FindRunningAgentStartPIDs lists PIDs whose argv contains consecutive "agent", "start"
// (typical `kernus agent start`). Excludes excludePID (usually the current `agent stop` process).
func FindRunningAgentStartPIDs(excludePID int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == excludePID {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil || len(data) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
		if matchesAgentStartArgs(args) {
			out = append(out, pid)
		}
	}
	return out, nil
}

func matchesAgentStartArgs(args []string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "agent" && args[i+1] == "start" {
			return true
		}
	}
	return false
}
