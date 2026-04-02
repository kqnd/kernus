package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Token    string `json:"token"`
}

func (c *Config) Validate() error {
	if c.Username == "" {
		return errors.New("username is required — run 'kernus config --username <user>'")
	}
	if c.Password == "" {
		return errors.New("password is required — run 'kernus config --password <pass>'")
	}
	return nil
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "kernus"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found — run 'kernus config' first")
		}
		return nil, fmt.Errorf("cannot open config: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}

	return &cfg, nil
}

func Save(cfg *Config) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("cannot marshal config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return "", fmt.Errorf("cannot write config: %w", err)
	}

	return path, nil
}
