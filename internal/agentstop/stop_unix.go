//go:build unix

package agentstop

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// StopPIDs asks each PID to terminate (SIGTERM), waits up to wait, then optionally SIGKILL survivors.
func StopPIDs(pids []int, wait time.Duration, force bool) error {
	for _, pid := range pids {
		p, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		_ = p.Signal(syscall.SIGTERM)
	}
	if len(pids) == 0 {
		return nil
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		anyAlive := false
		for _, pid := range pids {
			if processAlive(pid) {
				anyAlive = true
				break
			}
		}
		if !anyAlive {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	n := 0
	for _, pid := range pids {
		if processAlive(pid) {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("%d kernus agent process(es) still running after %v (try --force)", n, wait)
	}
	for _, pid := range pids {
		if !processAlive(pid) {
			continue
		}
		p, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		_ = p.Signal(syscall.SIGKILL)
	}
	return nil
}
