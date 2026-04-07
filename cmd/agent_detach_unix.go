//go:build unix

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// runDetached re-executes the agent without the --detach flag, fully detached from
// the terminal (new session, stdout/stderr redirected to the log file).
// kernus agent stop finds the background process via /proc cmdline scanning, so no
// PID file is needed — the existing stop command works as-is.
func runDetached(osArgs []string) error {
	logPath, err := agentLogPath()
	if err != nil {
		return fmt.Errorf("could not determine log path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return fmt.Errorf("could not create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open log file %s: %w", logPath, err)
	}
	defer logFile.Close()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	// Rebuild argv without the --detach / -d flag.
	var filteredArgs []string
	for _, a := range osArgs[1:] {
		if a == "--detach" || a == "-d" || strings.HasPrefix(a, "--detach=") {
			continue
		}
		filteredArgs = append(filteredArgs, a)
	}

	c := exec.Command(exe, filteredArgs...)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	c.Stdin = nil
	c.Stdout = logFile
	c.Stderr = logFile

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start agent in background: %w", err)
	}
	// Release the parent's reference so the child is not waited on.
	if err := c.Process.Release(); err != nil {
		return fmt.Errorf("could not release background process: %w", err)
	}

	fmt.Printf("✓ Agent started in background (PID %d)\n", c.Process.Pid)
	fmt.Printf("  Logs: %s\n", logPath)
	fmt.Printf("  Stop: kernus agent stop\n")
	return nil
}
