// Package agentlock ensures at most one Kernus agent process per agent token on this machine.
package agentlock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

const holdRetries = 40
const holdRetryDelay = 100 * time.Millisecond

func lockFilePath(token string) (path string, kernusDir string, err error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return "", "", fmt.Errorf("agent token is empty")
	}
	sum := sha256.Sum256([]byte(t))
	fp := hex.EncodeToString(sum[:16])
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	kd := filepath.Join(dir, "kernus")
	return filepath.Join(kd, "agent-"+fp+".lock"), kd, nil
}

// PreflightAvailable checks that no other agent holds the lock for this token.
// It acquires and immediately releases the lock so the child process can take it.
// Use before spawning a detached child to fail fast in the foreground terminal.
func PreflightAvailable(token string) error {
	path, kd, err := lockFilePath(token)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(kd, 0o700); err != nil {
		return fmt.Errorf("cannot create kernus config directory: %w", err)
	}
	fl := flock.New(path)
	ok, err := fl.TryLock()
	if err != nil {
		return fmt.Errorf("agent lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("another Kernus agent is already running with this agent token; stop it with: kernus agent stop")
	}
	// Close releases the exclusive lock and the file handle (see gofrs/flock Close/Unlock).
	return fl.Close()
}

// Hold acquires the exclusive lock for this token and returns a release function.
// It retries briefly to cover a small race after PreflightAvailable releases the lock.
func Hold(token string) (release func(), err error) {
	path, kd, err := lockFilePath(token)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(kd, 0o700); err != nil {
		return nil, fmt.Errorf("cannot create kernus config directory: %w", err)
	}
	fl := flock.New(path)
	for attempt := 0; attempt < holdRetries; attempt++ {
		ok, tryErr := fl.TryLock()
		if tryErr != nil {
			return nil, fmt.Errorf("agent lock: %w", tryErr)
		}
		if ok {
			return func() {
				_ = fl.Close()
			}, nil
		}
		time.Sleep(holdRetryDelay)
	}
	return nil, fmt.Errorf("another Kernus agent is already running with this agent token; stop it with: kernus agent stop")
}
