package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var ErrNoSession = errors.New("not logged in — run 'kernus login'")

type StoredSession struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

func sessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "kernus", "session.json"), nil
}

func SaveSession(s *StoredSession) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return fmt.Errorf("cannot create session directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal session: %w", err)
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("cannot write session: %w", err)
	}

	return nil
}

func LoadSession() (*StoredSession, error) {
	path, err := sessionPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("cannot open session: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read session: %w", err)
	}

	var s StoredSession
	err = json.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("invalid session format: %w", err)
	}

	return &s, nil
}

func DeleteSession() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete session: %w", err)
	}

	return nil
}

func IsSessionValid() bool {
	s, err := LoadSession()
	if err != nil {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}
