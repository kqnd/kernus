package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kiev/kernus/internal/agent"
	"github.com/kiev/kernus/internal/config"
	"github.com/spf13/cobra"
)

const (
	maxConsecutiveErrors = 10
	baseBackoff          = 5 * time.Second
	maxBackoff           = 5 * time.Minute
	dockerReconnectDelay = 10 * time.Second
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run metrics agent",
}

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the docker metrics agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		runtimeCfg, err := config.ResolveAgentRuntimeConfig()
		if err != nil {
			return err
		}

		if strings.TrimSpace(runtimeCfg.ServerURL) == "" {
			return fmt.Errorf("missing KERNUS_SERVER_URL (env or 'kernus token ... --server')")
		}
		if strings.TrimSpace(runtimeCfg.AgentToken) == "" {
			return fmt.Errorf("missing KERNUS_AGENT_TOKEN (env or 'kernus token <token>')")
		}
		if strings.TrimSpace(runtimeCfg.HostName) == "" {
			hn, _ := os.Hostname()
			runtimeCfg.HostName = hn
		}
		if runtimeCfg.Interval <= 0 {
			runtimeCfg.Interval = 30
		}

		dockerHost, err := cmd.Flags().GetString("docker-host")
		if err != nil {
			return err
		}

		useMock, err := cmd.Flags().GetBool("mock")
		if err != nil {
			return err
		}

		sender := agent.NewSender(runtimeCfg.ServerURL, runtimeCfg.AgentToken)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		serverCfg, fetchErr := sender.FetchConfig(ctx)
		if fetchErr != nil {
			fmt.Printf("⚠ Could not fetch server config: %v (using local interval %ds)\n", fetchErr, runtimeCfg.Interval)
		} else {
			if serverCfg.CollectionIntervalSeconds > 0 {
				runtimeCfg.Interval = serverCfg.CollectionIntervalSeconds
			}
		}

		fmt.Printf("✓ Agent started. Collecting metrics every %ds.\n", runtimeCfg.Interval)
		fmt.Printf("✓ Connected to %s\n", runtimeCfg.ServerURL)
		fmt.Printf("✓ Host: %s\n", runtimeCfg.HostName)
		if fetchErr == nil {
			orgName := serverCfg.OrgName
			if orgName == "" {
				orgName = "(unknown)"
			}
			planName := serverCfg.PlanName
			if planName == "" {
				planName = "(unknown)"
			}
			fmt.Printf("✓ Org: %s\n", orgName)
			fmt.Printf("✓ Plan: %s\n", planName)
		}
		if useMock {
			fmt.Println("✓ Mock mode: using simulated containers with extreme behaviors")
		}

		if !useMock && fetchErr == nil {
			pfCollector, pfErr := agent.NewCollector(dockerHost)
			if pfErr == nil {
				localCount, _ := pfCollector.CountContainers(ctx)
				pfCollector.Close()

				pf, pfErr := sender.Preflight(ctx, runtimeCfg.HostName, localCount)
				if pfErr == nil && !pf.CanProceed {
					maxC := fmt.Sprintf("%d", pf.MaxContainers)
					if pf.MaxContainers == -1 {
						maxC = "unlimited"
					}
					maxH := fmt.Sprintf("%d", pf.MaxHosts)
					if pf.MaxHosts == -1 {
						maxH = "unlimited"
					}
					fmt.Printf("✗ Cannot start monitoring: %s\n", pf.Reason)
					fmt.Printf("  Containers: %d/%s  |  Hosts: %d/%s\n",
						pf.CurrentContainers, maxC, pf.CurrentHosts, maxH)
					return nil
				}
			}
		}

		for ctx.Err() == nil {
			err := runAgentLoop(ctx, stop, runtimeCfg, dockerHost, sender, useMock)
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				fmt.Printf("✗ Agent loop exited with error: %v\n", err)
				fmt.Printf("→ Reconnecting in %s...\n", dockerReconnectDelay)
				select {
				case <-ctx.Done():
				case <-time.After(dockerReconnectDelay):
				}
			}
		}

		fmt.Println("✓ Agent stopped")
		return nil
	},
}

func runAgentLoop(ctx context.Context, stop context.CancelFunc, runtimeCfg *config.AgentConfig, dockerHost string, sender *agent.Sender, useMock bool) error {
	var collector agent.MetricCollector
	if useMock {
		collector = agent.NewMockCollector()
	} else {
		realCollector, err := agent.NewCollector(dockerHost)
		if err != nil {
			return fmt.Errorf("docker connection failed: %w", err)
		}
		collector = realCollector
	}
	defer collector.Close()

	consecutiveErrors := 0
	configRefreshCounter := 0
	const configRefreshInterval = 30

	// Track previous container states to detect critical transitions
	prevStates := make(map[string]string) // containerID → status

	runCycle := func(cycleCtx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("✗ PANIC recovered in collect cycle: %v\n", r)
				consecutiveErrors++
			}
		}()

		containers, collectErr := collector.Collect(cycleCtx)
		if collectErr != nil {
			consecutiveErrors++
			fmt.Printf("✗ collect error (%d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, collectErr)
			return
		}
		if len(containers) == 0 {
			consecutiveErrors = 0
			return
		}

		// Detect critical transitions and capture log snapshots
		var snapshots []agent.LogSnapshot
		currentStates := make(map[string]string, len(containers))
		for _, c := range containers {
			currentStates[c.ID] = c.Status
			prev, existed := prevStates[c.ID]
			if !existed {
				continue
			}
			// Critical: was running, now exited/dead with non-zero exit or OOM
			if prev == "running" && (c.Status == "exited" || c.Status == "dead") && (c.ExitCode != 0 || c.OOMKilled) {
				eventType := "crash"
				if c.OOMKilled {
					eventType = "oom_kill"
				}
				logs, logErr := collector.GetContainerLogs(cycleCtx, c.ID, 100)
				if logErr != nil {
					fmt.Printf("⚠ Could not capture logs for %s: %v\n", c.Name, logErr)
					logs = nil
				}
				snapshots = append(snapshots, agent.LogSnapshot{
					ContainerID:   c.ID,
					ContainerName: c.Name,
					Timestamp:     c.Timestamp,
					EventType:     eventType,
					ExitCode:      c.ExitCode,
					ExitReason:    c.ExitReason,
					LogLines:      logs,
					CPUPercent:    c.CPUPercent,
					MemoryUsed:    c.MemoryUsed,
					MemoryLimit:   c.MemoryLimit,
				})
			}
		}
		prevStates = currentStates

		// Send log snapshots (non-blocking: don't fail the whole cycle)
		if len(snapshots) > 0 {
			go func() {
				snapCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := sender.SendLogSnapshots(snapCtx, agent.LogSnapshotRequest{
					HostName:  runtimeCfg.HostName,
					SentAt:    time.Now().UTC(),
					Snapshots: snapshots,
				}); err != nil {
					fmt.Printf("⚠ Failed to send log snapshots: %v\n", err)
				}
			}()
		}

		for start := 0; start < len(containers); start += 500 {
			end := start + 500
			if end > len(containers) {
				end = len(containers)
			}
			batch := containers[start:end]
			payload := agent.IngestRequest{
				HostName:   runtimeCfg.HostName,
				SentAt:     time.Now().UTC(),
				Containers: batch,
			}
			actionErr := sendWithPolicy(cycleCtx, sender, payload)
			if actionErr != nil {
				if actionErr == errInvalidAgentToken {
					fmt.Println("✗ Agent token invalid — stopping agent")
					stop()
					return
				}
				consecutiveErrors++
				fmt.Printf("✗ ingest error (%d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, actionErr)
				return
			}
		}
		consecutiveErrors = 0
	}

	for {
		if ctx.Err() != nil {
			break
		}

		if consecutiveErrors >= maxConsecutiveErrors {
			fmt.Printf("✗ %d consecutive errors — forcing Docker reconnect\n", consecutiveErrors)
			return fmt.Errorf("too many consecutive errors (%d)", consecutiveErrors)
		}

		configRefreshCounter++
		if configRefreshCounter%configRefreshInterval == 0 {
			if serverCfg, fetchErr := sender.FetchConfig(ctx); fetchErr == nil && serverCfg.CollectionIntervalSeconds > 0 {
				if runtimeCfg.Interval != serverCfg.CollectionIntervalSeconds {
					fmt.Printf("→ Plan collection interval updated: %ds (was %ds)\n", serverCfg.CollectionIntervalSeconds, runtimeCfg.Interval)
					runtimeCfg.Interval = serverCfg.CollectionIntervalSeconds
				}
			}
		}

		t0 := time.Now()
		runCycle(ctx)
		if ctx.Err() != nil {
			break
		}

		interval := time.Duration(runtimeCfg.Interval) * time.Second
		if consecutiveErrors > 0 {
			backoff := baseBackoff * time.Duration(1<<uint(consecutiveErrors-1))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			interval = backoff
			fmt.Printf("→ Backing off for %s due to errors\n", interval)
		}

		nextTick := t0.Add(interval)
		wait := time.Until(nextTick)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			fmt.Println("→ Graceful shutdown: executing final collection cycle...")
			finalCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			runCycle(finalCtx)
			cancel()
			return nil
		case <-timer.C:
		}
	}

	return nil
}

var errInvalidAgentToken = fmt.Errorf("invalid agent token")

func sendWithPolicy(ctx context.Context, sender *agent.Sender, payload agent.IngestRequest) error {
	for attempt := 0; attempt < 4; attempt++ {
		result, err := sender.Send(ctx, payload)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		switch result.StatusCode {
		case 202:
			return nil
		case 400:
			return fmt.Errorf("payload invalido: %s", result.Body)
		case 401:
			return errInvalidAgentToken
		case 402:
			fmt.Printf("⚠ Plan limit reached: %s\n", result.Body)
			return nil
		case 429:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(60 * time.Second):
			}
			continue
		default:
			if result.StatusCode >= 500 && result.StatusCode <= 599 {
				if attempt >= 2 {
					return fmt.Errorf("server error %d: %s", result.StatusCode, result.Body)
				}
				backoff := time.Duration(5*(1<<attempt)) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			return fmt.Errorf("unexpected status %d: %s", result.StatusCode, result.Body)
		}
	}
	return nil
}

func init() {
	agentStartCmd.Flags().String("docker-host", "", "Docker daemon URL (optional)")
	agentStartCmd.Flags().Bool("mock", false, "Use mock containers with extreme random behaviors (testing)")
	agentCmd.AddCommand(agentStartCmd)
	rootCmd.AddCommand(agentCmd)
}
