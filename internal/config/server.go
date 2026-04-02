package config

import (
	"os"
	"strings"
)

const DefaultServerURL = "https://api.kernus.app"

func ResolveServerURL(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return strings.TrimSuffix(v, "/")
	}

	if v := strings.TrimSpace(os.Getenv("KERNUS_SERVER_URL")); v != "" {
		return strings.TrimSuffix(v, "/")
	}

	if agentCfg, err := LoadAgentConfig(); err == nil {
		if v := strings.TrimSpace(agentCfg.ServerURL); v != "" {
			return strings.TrimSuffix(v, "/")
		}
	}

	if cfg, err := Load(); err == nil {
		if v := strings.TrimSpace(cfg.Server); v != "" {
			return strings.TrimSuffix(v, "/")
		}
	}

	return DefaultServerURL
}
