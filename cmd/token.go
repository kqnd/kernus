package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kiev/kernus/internal/agent"
	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/config"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token [agent-token]",
	Short: "Create or configure agent token",
	Long: `Manage Agent Token lifecycle.

Examples:
  kernus token create "prod-server-01"
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
			fmt.Printf("✓ Interval: %ds (plan)\n", serverCfg.CollectionIntervalSeconds)
		} else {
			fmt.Printf("✓ Interval: %ds (local default — plan will be applied at startup)\n", cfg.Interval)
		}
		return nil
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create an agent token for the current organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := auth.LoadSession()
		if err != nil {
			return fmt.Errorf("not logged in — run 'kernus login' first")
		}
		if !auth.IsSessionValid() {
			return fmt.Errorf("session has expired — run 'kernus login' again")
		}

		orgID := strings.TrimSpace(session.OrgID)
		if orgID == "" {
			orgID, err = auth.ExtractOrgID(session.Token)
			if err != nil {
				return fmt.Errorf("cannot determine org_id from access token: %w", err)
			}
		}

		serverURL, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}
		serverURL = config.ResolveServerURL(serverURL)

		type createReq struct {
			Name string `json:"name"`
		}
		type createResp struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Token string `json:"token"`
		}

		body, err := json.Marshal(createReq{Name: strings.TrimSpace(args[0])})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		url := strings.TrimSuffix(serverURL, "/") + "/v1/orgs/" + orgID + "/agent-tokens"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+session.Token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create token (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}

		var parsed createResp
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return fmt.Errorf("invalid create token response: %w", err)
		}
		if parsed.Token == "" {
			return fmt.Errorf("create token response missing token")
		}

		fmt.Println("✓ Agent token created:")
		fmt.Println()
		fmt.Printf("  %s\n", parsed.Token)
		fmt.Println()
		fmt.Println("  ⚠ Guarde este token agora. Ele nao sera exibido novamente.")
		fmt.Println()
		fmt.Println("Proximo passo:")
		fmt.Printf("  kernus token %s --server %s --host %s\n", parsed.Token, strings.TrimSuffix(serverURL, "/"), parsed.Name)
		return nil
	},
}

func init() {
	tokenCmd.Flags().String("server", "", "Kernus API server URL (precedence: flag > env > config > default)")
	tokenCmd.Flags().String("host", "", "Host name to report")
	tokenCmd.Flags().Int("interval", 30, "Collection interval in seconds")

	tokenCreateCmd.Flags().String("server", "", "Kernus API server URL (precedence: flag > env > config > default)")
	tokenCmd.AddCommand(tokenCreateCmd)
	rootCmd.AddCommand(tokenCmd)
}
