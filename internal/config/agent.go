package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type AgentConfig struct {
	ServerURL  string `json:"server_url"`
	AgentToken string `json:"agent_token"`
	HostName   string `json:"host_name"`
	Interval   int    `json:"interval"`
}

func agentConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "kernus", "agent.conf"), nil
}

func LoadAgentConfig() (*AgentConfig, error) {
	path, err := agentConfigPath()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AgentConfig{}, nil
		}
		return nil, fmt.Errorf("cannot read agent config: %w", err)
	}

	var cfg AgentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("invalid agent config format: %w", err)
	}

	return &cfg, nil
}

func SaveAgentConfig(cfg *AgentConfig) (string, error) {
	path, err := agentConfigPath()
	if err != nil {
		return "", err
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 30
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("cannot encode agent config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("cannot write agent config: %w", err)
	}

	return path, nil
}

func ResolveAgentRuntimeConfig() (*AgentConfig, error) {
	cfg, err := LoadAgentConfig()
	if err != nil {
		return nil, err
	}

	if v := strings.TrimSpace(os.Getenv("KERNUS_SERVER_URL")); v != "" {
		cfg.ServerURL = v
	}
	if v := strings.TrimSpace(os.Getenv("KERNUS_AGENT_TOKEN")); v != "" {
		cfg.AgentToken = v
	}
	if v := strings.TrimSpace(os.Getenv("KERNUS_HOST_NAME")); v != "" {
		cfg.HostName = v
	}
	if v := strings.TrimSpace(os.Getenv("KERNUS_INTERVAL")); v != "" {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid KERNUS_INTERVAL value: %w", parseErr)
		}
		cfg.Interval = parsed
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 30
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		cfg.ServerURL = DefaultServerURL
	}

	return cfg, nil
}
