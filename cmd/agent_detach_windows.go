//go:build windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// runDetached re-executes the agent without the --detach flag as a detached
// Windows process (DETACHED_PROCESS), so it continues after the terminal closes.
// kernus agent stop finds it via PowerShell CIM cmdline matching.
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

	var filteredArgs []string
	for _, a := range osArgs[1:] {
		if a == "--detach" || a == "-d" || strings.HasPrefix(a, "--detach=") {
			continue
		}
		filteredArgs = append(filteredArgs, a)
	}

	c := exec.Command(exe, filteredArgs...)
	// DETACHED_PROCESS (0x00000008): child has no console and is not part of the
	// parent console group, so it keeps running after the terminal closes.
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008,
	}
	c.Stdin = nil
	c.Stdout = logFile
	c.Stderr = logFile

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start agent in background: %w", err)
	}
	if err := c.Process.Release(); err != nil {
		return fmt.Errorf("could not release background process: %w", err)
	}

	fmt.Printf("✓ Agent started in background (PID %d)\n", c.Process.Pid)
	fmt.Printf("  Logs: %s\n", logPath)
	fmt.Printf("  Stop: kernus agent stop\n")
	return nil
}
