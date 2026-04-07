package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kiev/kernus/internal/agent"
	"github.com/kiev/kernus/internal/config"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token [agent-token]",
	Short: "Configure agent token",
	Long: `Manage Agent Token lifecycle.

Examples:
  kernus token kn_live_a1b2c3... --server https://api.kernus.app --host prod-server-01`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("agent token is required")
		}

		token := strings.TrimSpace(args[0])
		if token == "" {
			return fmt.Errorf("agent token is required")
		}

		serverURL, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}
		hostName, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}
		interval, err := cmd.Flags().GetInt("interval")
		if err != nil {
			return err
		}

		serverURL = config.ResolveServerURL(serverURL)
		if strings.TrimSpace(hostName) == "" {
			hostName = os.Getenv("KERNUS_HOST_NAME")
		}
		if strings.TrimSpace(hostName) == "" {
			hn, _ := os.Hostname()
			hostName = hn
		}
		if interval <= 0 {
			interval = 30
		}

		cfg := &config.AgentConfig{
			ServerURL:  strings.TrimSpace(serverURL),
			AgentToken: token,
			HostName:   strings.TrimSpace(hostName),
			Interval:   interval,
		}
		path, err := config.SaveAgentConfig(cfg)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Token saved to %s\n", path)
		fmt.Printf("✓ Server URL: %s\n", cfg.ServerURL)
		fmt.Printf("✓ Host name: %s\n", cfg.HostName)

		sender := agent.NewSender(cfg.ServerURL, cfg.AgentToken)
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if serverCfg, err := sender.FetchConfig(ctx2); err == nil && serverCfg.CollectionIntervalSeconds > 0 {
			if cfg.Interval != serverCfg.CollectionIntervalSeconds {
				cfg.Interval = serverCfg.CollectionIntervalSeconds
				if _, saveErr := config.SaveAgentConfig(cfg); saveErr != nil {
					fmt.Printf("⚠ Could not persist plan interval locally: %v\n", saveErr)
				}
			}
			fmt.Printf("✓ Interval: %ds (plan)\n", serverCfg.CollectionIntervalSeconds)
		} else {
			fmt.Printf("✓ Interval: %ds (local default — plan will be applied at startup)\n", cfg.Interval)
		}
		return nil
	},
}

func init() {
	tokenCmd.Flags().String("server", "", "Kernus API server URL (precedence: flag > env > config > default)")
	tokenCmd.Flags().String("host", "", "Host name to report")
	tokenCmd.Flags().Int("interval", 30, "Collection interval in seconds")
	rootCmd.AddCommand(tokenCmd)
}
