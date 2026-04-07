package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kiev/kernus/internal/agent"
	"github.com/kiev/kernus/internal/config"
	"github.com/kiev/kernus/internal/update"
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

// agentLogPath returns the path for the background agent log file.
func agentLogPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kernus", "agent.log"), nil
}

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the docker metrics agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Optional one-shot config flags: allow `kernus agent start --token ...` without a prior `kernus token ...`.
		// If provided, we persist them to agent.conf so future starts work too (including detached runs).
		tokenFlag, _ := cmd.Flags().GetString("token")
		serverFlag, _ := cmd.Flags().GetString("server")
		hostFlag, _ := cmd.Flags().GetString("host")
		intervalFlag, _ := cmd.Flags().GetInt("interval")
		if intervalFlag < 0 {
			intervalFlag = 0
		}

		detach, _ := cmd.Flags().GetBool("detach")
		if detach {
			return runDetached(os.Args)
		}

		updateCtx, cancelUpdate := context.WithTimeout(context.Background(), 30*time.Second)
		latestVersion, updateErr := update.NewClient("").MaybeSelfUpdate(updateCtx, currentVersion)
		cancelUpdate()
		if updateErr != nil {
			if errors.Is(updateErr, update.ErrRestartScheduled) {
				fmt.Printf("→ Updated Kernus agent to %s. Restarting with the new binary...\n", latestVersion)
				return nil
			}
			fmt.Printf("⚠ Auto-update skipped: %v\n", updateErr)
		} else if latestVersion != "" {
			// On Unix the process re-execs before returning here. This log is only a fallback.
			fmt.Printf("→ Updated Kernus agent to %s. Restarting with the new binary...\n", latestVersion)
			return nil
		}

		runtimeCfg, err := config.ResolveAgentRuntimeConfig()
		if err != nil {
			return err
		}

		if strings.TrimSpace(tokenFlag) != "" || strings.TrimSpace(serverFlag) != "" || strings.TrimSpace(hostFlag) != "" || intervalFlag > 0 {
			if strings.TrimSpace(tokenFlag) != "" {
				runtimeCfg.AgentToken = strings.TrimSpace(tokenFlag)
			}
			if strings.TrimSpace(serverFlag) != "" {
				runtimeCfg.ServerURL = config.ResolveServerURL(strings.TrimSpace(serverFlag))
			}
			if strings.TrimSpace(hostFlag) != "" {
				runtimeCfg.HostName = strings.TrimSpace(hostFlag)
			}
			if intervalFlag > 0 {
				runtimeCfg.Interval = intervalFlag
			}
			if _, saveErr := config.SaveAgentConfig(runtimeCfg); saveErr != nil {
				return fmt.Errorf("cannot persist agent config: %w", saveErr)
			}
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

		dockerList := resolveDockerListOptions(cmd)

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
				persistPlanInterval(serverCfg.CollectionIntervalSeconds)
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

		if !useMock {
			switch {
			case dockerList.AllContainers:
				fmt.Println("✓ Container scope: all states (docker ps -a), including stopped/exited")
			case len(dockerList.NamePrefixes) > 0:
				fmt.Printf("✓ Container scope: running containers matching name prefix %v\n", dockerList.NamePrefixes)
			default:
				fmt.Println("✓ Container scope: running only (matches docker ps; stopped containers are not counted)")
			}
		}

		if !useMock && fetchErr == nil {
			pfCollector, pfErr := agent.NewCollector(dockerHost, dockerList)
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
					if pf.ContainersThisHost+pf.ContainersOtherHosts == pf.CurrentContainers && pf.CurrentContainers > 0 {
						fmt.Printf("  → This host would report %d container(s); %d distinct ID(s) already counted from other host(s) in the org (last 24h).\n",
							pf.ContainersThisHost, pf.ContainersOtherHosts)
					}
					return nil
				}
			}
		}

		for ctx.Err() == nil {
			err := runAgentLoop(ctx, stop, runtimeCfg, dockerHost, sender, useMock, dockerList)
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

// tombstoneMetric synthesizes a final row when a container disappears from Docker between
// collection cycles. We do not assume clean_stop — real cause was not observed via inspect.
func tombstoneMetric(prev agent.ContainerMetric, ts time.Time) agent.ContainerMetric {
	m := prev
	m.Timestamp = ts
	m.Status = "exited"
	m.Health = "none"
	m.ExitCode = 0
	m.ExitReason = "unknown"
	m.OOMKilled = false
	m.CPUPercent = 0
	m.MemoryUsed = 0
	m.NetworkRxBytes = 0
	m.NetworkTxBytes = 0
	return m
}

// inferRemovedSnapshotFromLogs classifies log tail when the container ID is already gone from Docker.
// Docker may still return recent logs briefly after stop/remove.
func inferRemovedSnapshotFromLogs(logs []string) (eventType string, exitReason string) {
	eventType = "removed"
	exitReason = "unknown"
	if len(logs) == 0 {
		return
	}
	var b strings.Builder
	for _, line := range logs {
		b.WriteString(strings.ToLower(line))
		b.WriteByte('\n')
	}
	j := b.String()
	if strings.Contains(j, "oomkilled") || strings.Contains(j, "out of memory") || strings.Contains(j, "cannot allocate memory") {
		return "oom_kill", "oom_killed"
	}
	if strings.Contains(j, "panic:") || strings.Contains(j, "fatal error") || strings.Contains(j, "segmentation fault") {
		return "crash", "app_crashed"
	}
	return
}

func applyTombstoneHints(m *agent.ContainerMetric, exitReason string) {
	switch exitReason {
	case "oom_killed":
		m.ExitReason = "oom_killed"
		m.OOMKilled = true
		m.ExitCode = 137
	case "app_crashed":
		m.ExitReason = "app_crashed"
		m.ExitCode = 1
	default:
		// keep unknown / 0 from tombstoneMetric
	}
}

func runAgentLoop(ctx context.Context, stop context.CancelFunc, runtimeCfg *config.AgentConfig, dockerHost string, sender *agent.Sender, useMock bool, dockerList agent.DockerListOptions) error {
	var collector agent.MetricCollector
	if useMock {
		collector = agent.NewMockCollector()
	} else {
		realCollector, err := agent.NewCollector(dockerHost, dockerList)
		if err != nil {
			return fmt.Errorf("docker connection failed: %w", err)
		}
		collector = realCollector
	}
	defer collector.Close()

	// Start Docker event watcher in a background goroutine.
	// On failure (e.g. mock mode or Docker unreachable) we fall back gracefully.
	var watcher *agent.EventWatcher
	if !useMock {
		w, watchErr := agent.NewEventWatcher(dockerHost)
		if watchErr != nil {
			fmt.Printf("⚠ Docker event watcher unavailable: %v (tombstones will use log heuristics)\n", watchErr)
		} else {
			watcher = w
			go watcher.Run(ctx)
			defer watcher.Close()
		}
	}

	consecutiveErrors := 0
	lastConfigRefresh := time.Now()
	const configRefreshMinInterval = 1 * time.Minute

	// Log snapshot cooldown: prevents snapshot storms when a container is in a crash loop.
	// After the first snapshot, we suppress further snapshots for the same container
	// until snapshotCooldown has elapsed. The cooldown resets naturally once the container
	// is healthy long enough for the timer to expire.
	const snapshotCooldown = 15 * time.Minute
	lastSnapshot := make(map[string]time.Time) // containerID → last snapshot sent time

	canSendSnapshot := func(containerID string) bool {
		last, ok := lastSnapshot[containerID]
		return !ok || time.Since(last) >= snapshotCooldown
	}
	markSnapshot := func(containerID string) {
		lastSnapshot[containerID] = time.Now()
	}

	// Previous cycle: full metric per ID (for tombstones) and status (for transitions).
	prevStates := make(map[string]string) // containerID → status
	prevByID := make(map[string]agent.ContainerMetric)

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

		if len(containers) == 0 && len(prevByID) == 0 {
			consecutiveErrors = 0
			return
		}

		currentByID := make(map[string]struct{}, len(containers))
		currentStates := make(map[string]string, len(containers))
		for _, c := range containers {
			currentByID[c.ID] = struct{}{}
			currentStates[c.ID] = c.Status
		}

		ts := time.Now().UTC()
		var snapshots []agent.LogSnapshot
		tombstones := make([]agent.ContainerMetric, 0)

		// Disappeared from Docker since last cycle → synthetic exited + snapshot.
		// Prefer event-watcher data (captured at die time with real exit code + fresh logs).
		// Fall back to log heuristics if the watcher was unavailable or missed the event.
		// Tombstones always bypass the snapshot cooldown: container removal is a terminal event,
		// not a repeating one, so we always want one final record.
		for _, prev := range prevByID {
			if _, stillThere := currentByID[prev.ID]; stillThere {
				continue
			}

			tmb := tombstoneMetric(prev, ts)
			var evType, evReason string
			var logLines []string

			if watcher != nil {
				if cap := watcher.Drain(prev.ID); cap != nil {
					// Watcher captured precise exit info at the moment of death.
					tmb.ExitCode = cap.ExitCode
					tmb.OOMKilled = cap.OOMKilled
					tmb.ExitReason = cap.ExitReason
					logLines = cap.LogLines
					if cap.MemoryLimit > 0 {
						tmb.MemoryLimit = cap.MemoryLimit
					}
					evReason = cap.ExitReason
					switch {
					case cap.OOMKilled:
						evType = "oom_kill"
					case cap.ExitCode != 0:
						evType = "crash"
					default:
						evType = "clean_stop"
					}
				}
			}

			if evType == "" {
				// Fallback: try docker logs on the old ID (may be empty if already removed).
				logs, logErr := collector.GetContainerLogs(cycleCtx, prev.ID, 100)
				if logErr != nil {
					logs = nil
				}
				logLines = logs
				evType, evReason = inferRemovedSnapshotFromLogs(logs)
				applyTombstoneHints(&tmb, evReason)
			}

			tombstones = append(tombstones, tmb)
			snapshots = append(snapshots, agent.LogSnapshot{
				ContainerID:   prev.ID,
				ContainerName: prev.Name,
				Timestamp:     ts,
				EventType:     evType,
				ExitCode:      tmb.ExitCode,
				ExitReason:    tmb.ExitReason,
				LogLines:      logLines,
				CPUPercent:    0,
				MemoryUsed:    0,
				MemoryLimit:   tmb.MemoryLimit,
			})
			markSnapshot(prev.ID)
		}

		// In-list transitions: running → exited/dead (logs when container still exists).
		for _, c := range containers {
			prev, existed := prevStates[c.ID]
			if !existed {
				continue
			}
			if prev == "running" && (c.Status == "exited" || c.Status == "dead") {
				if !canSendSnapshot(c.ID) {
					continue
				}
				eventType := "clean_stop"
				switch {
				case c.OOMKilled:
					eventType = "oom_kill"
				case c.Status == "dead" || c.ExitCode != 0:
					eventType = "crash"
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
				markSnapshot(c.ID)
			}
		}

		// Restart count increased since last cycle (crash loop / policy restart).
		// Docker often reports "running" again with a higher restart_count before we see exited in a poll,
		// so without this we never attach logs to restart storms.
		// Rate-limited by snapshotCooldown: we capture the first occurrence and then back off.
		for _, c := range containers {
			prev, ok := prevByID[c.ID]
			if !ok || c.RestartCount <= prev.RestartCount {
				continue
			}
			if !canSendSnapshot(c.ID) {
				continue
			}
			logs, logErr := collector.GetContainerLogs(cycleCtx, c.ID, 100)
			if logErr != nil {
				fmt.Printf("⚠ Could not capture logs after restart for %s: %v\n", c.Name, logErr)
				logs = nil
			}
			evType := "crash"
			if c.OOMKilled {
				evType = "oom_kill"
			}
			ec := c.ExitCode
			er := c.ExitReason
			if c.Status == "running" || c.Status == "restarting" {
				if er == "" || er == "clean_stop" || er == "unknown" {
					er = "app_crashed"
				}
				if ec == 0 {
					ec = 1
				}
			}
			snapshots = append(snapshots, agent.LogSnapshot{
				ContainerID:   c.ID,
				ContainerName: c.Name,
				Timestamp:     ts,
				EventType:     evType,
				ExitCode:      ec,
				ExitReason:    er,
				LogLines:      logs,
				CPUPercent:    c.CPUPercent,
				MemoryUsed:    c.MemoryUsed,
				MemoryLimit:   c.MemoryLimit,
			})
			markSnapshot(c.ID)
		}

		toSend := make([]agent.ContainerMetric, 0, len(containers)+len(tombstones))
		toSend = append(toSend, containers...)
		toSend = append(toSend, tombstones...)

		if len(toSend) == 0 {
			consecutiveErrors = 0
			return
		}

		// Send log snapshots (non-blocking: don't fail the whole cycle)
		if len(snapshots) > 0 {
			go func(snaps []agent.LogSnapshot) {
				snapCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := sender.SendLogSnapshots(snapCtx, agent.LogSnapshotRequest{
					HostName:  runtimeCfg.HostName,
					SentAt:    time.Now().UTC(),
					Snapshots: snaps,
				}); err != nil {
					fmt.Printf("⚠ Failed to send log snapshots: %v\n", err)
				}
			}(snapshots)
		}

		for start := 0; start < len(toSend); start += 500 {
			end := start + 500
			if end > len(toSend) {
				end = len(toSend)
			}
			batch := toSend[start:end]
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

		nextPrev := make(map[string]agent.ContainerMetric, len(containers))
		for _, c := range containers {
			nextPrev[c.ID] = c
		}
		prevByID = nextPrev
		prevStates = currentStates
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

		if time.Since(lastConfigRefresh) >= configRefreshMinInterval {
			if serverCfg, fetchErr := sender.FetchConfig(ctx); fetchErr == nil && serverCfg.CollectionIntervalSeconds > 0 {
				if runtimeCfg.Interval != serverCfg.CollectionIntervalSeconds {
					fmt.Printf("→ Plan collection interval updated: %ds (was %ds)\n", serverCfg.CollectionIntervalSeconds, runtimeCfg.Interval)
					runtimeCfg.Interval = serverCfg.CollectionIntervalSeconds
					persistPlanInterval(serverCfg.CollectionIntervalSeconds)
				}
			}
			lastConfigRefresh = time.Now()
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

func persistPlanInterval(interval int) {
	if interval <= 0 || strings.TrimSpace(os.Getenv("KERNUS_INTERVAL")) != "" {
		return
	}
	cfg, err := config.LoadAgentConfig()
	if err != nil || cfg == nil {
		return
	}
	if cfg.AgentToken == "" || cfg.ServerURL == "" || cfg.Interval == interval {
		return
	}
	cfg.Interval = interval
	if _, err := config.SaveAgentConfig(cfg); err != nil {
		fmt.Printf("⚠ Could not persist plan interval locally: %v\n", err)
	}
}

func resolveDockerListOptions(cmd *cobra.Command) agent.DockerListOptions {
	all, _ := cmd.Flags().GetBool("all-containers")
	prefixes, _ := cmd.Flags().GetStringSlice("name-prefix")
	if v := strings.TrimSpace(os.Getenv("KERNUS_AGENT_ALL_CONTAINERS")); v == "1" || strings.EqualFold(v, "true") {
		all = true
	}
	if v := strings.TrimSpace(os.Getenv("KERNUS_AGENT_NAME_PREFIX")); v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				prefixes = append(prefixes, p)
			}
		}
	}
	return agent.DockerListOptions{AllContainers: all, NamePrefixes: prefixes}
}

func init() {
	agentStartCmd.Flags().String("docker-host", "", "Docker daemon URL (optional)")
	agentStartCmd.Flags().Bool("all-containers", false, "List stopped/exited containers too (docker ps -a); counts against plan limits")
	agentStartCmd.Flags().StringSlice("name-prefix", nil, "Only monitor containers whose name starts with this prefix (repeat flag for multiple). Implies running-only unless --all-containers")
	agentStartCmd.Flags().Bool("mock", false, "Use mock containers with extreme random behaviors (testing)")
	agentStartCmd.Flags().BoolP("detach", "d", true, "Run agent in the background (detach from terminal); logs written to the kernus config directory (use --detach=false to run in foreground)")
	agentStartCmd.Flags().String("token", "", "Agent token (kn_live_...). If set, saves it to agent.conf before starting")
	agentStartCmd.Flags().String("server", "", "Kernus API server URL. If set, saves it to agent.conf before starting")
	agentStartCmd.Flags().String("host", "", "Host name to report. If set, saves it to agent.conf before starting")
	agentStartCmd.Flags().Int("interval", 0, "Collection interval in seconds. If set (>0), saves it to agent.conf before starting")
	agentCmd.AddCommand(agentStartCmd)
	rootCmd.AddCommand(agentCmd)
}
