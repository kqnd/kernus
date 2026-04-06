package agent

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// CapturedExit holds everything collected at the exact moment Docker fires the die event.
// It is consumed by the next ingest cycle to build a precise tombstone/snapshot.
type CapturedExit struct {
	ContainerID   string
	ContainerName string
	ExitCode      int32
	OOMKilled     bool
	ExitReason    string
	LogLines      []string
	MemoryLimit   uint64
	CapturedAt    time.Time
}

// EventWatcher subscribes to Docker container die/oom events and captures exit context
// (exit code + logs) immediately while the container still exists in the exited state.
//
// Call Run in a goroutine. It reconnects automatically if the stream drops.
// Call Drain(containerID) from the polling cycle to consume a captured exit.
type EventWatcher struct {
	cli *client.Client

	mu      sync.Mutex
	pending map[string]*CapturedExit // containerID → captured context
	oomSeen map[string]struct{}       // containerIDs that had an oom event before die
}

// NewEventWatcher creates a watcher that connects to the same Docker host as the Collector.
func NewEventWatcher(dockerHost string) (*EventWatcher, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if strings.TrimSpace(dockerHost) != "" {
		opts = append(opts, client.WithHost(strings.TrimSpace(dockerHost)))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("event watcher: cannot create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("event watcher: cannot reach Docker daemon: %w", err)
	}

	return &EventWatcher{
		cli:     cli,
		pending: make(map[string]*CapturedExit),
		oomSeen: make(map[string]struct{}),
	}, nil
}

// Close shuts down the underlying Docker client.
func (w *EventWatcher) Close() error {
	return w.cli.Close()
}

// Drain returns and removes the captured exit for a container ID, or nil if none.
func (w *EventWatcher) Drain(containerID string) *CapturedExit {
	w.mu.Lock()
	defer w.mu.Unlock()
	cap := w.pending[containerID]
	delete(w.pending, containerID)
	return cap
}

// Run subscribes to Docker events and blocks until ctx is cancelled.
// It auto-reconnects on stream errors so the caller only needs one goroutine.
func (w *EventWatcher) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := w.watch(ctx); err != nil && ctx.Err() == nil {
			fmt.Printf("⚠ Docker event stream error: %v — reconnecting in 5s\n", err)
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (w *EventWatcher) watch(ctx context.Context) error {
	f := filters.NewArgs()
	f.Add("type", string(events.ContainerEventType))
	f.Add("event", string(events.ActionDie))
	f.Add("event", string(events.ActionOOM))

	msgCh, errCh := w.cli.Events(ctx, events.ListOptions{Filters: f})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case msg := <-msgCh:
			// OOM fires before die; mark it so the die handler can detect it.
			if msg.Action == events.ActionOOM {
				w.mu.Lock()
				w.oomSeen[msg.Actor.ID] = struct{}{}
				w.mu.Unlock()
				continue
			}
			// die event — capture in a goroutine so we don't block the event stream.
			go w.handleDie(ctx, msg)
		}
	}
}

func (w *EventWatcher) handleDie(ctx context.Context, msg events.Message) {
	containerID := msg.Actor.ID
	name := msg.Actor.Attributes["name"]
	if name == "" {
		n := containerID
		if len(n) > 12 {
			n = n[:12]
		}
		name = n
	}

	// Check if an OOM event preceded this die.
	w.mu.Lock()
	_, oomKilled := w.oomSeen[containerID]
	delete(w.oomSeen, containerID)
	w.mu.Unlock()

	var exitCode int32
	if ecStr, ok := msg.Actor.Attributes["exitCode"]; ok {
		if ec, err := strconv.ParseInt(ecStr, 10, 32); err == nil {
			exitCode = int32(ec)
		}
	}
	if oomKilled && exitCode == 0 {
		exitCode = 137
	}

	exitReason := ClassifyExitReason(int(exitCode), oomKilled)

	// Fetch logs now — container is in exited state, not yet removed, logs are still available.
	var logLines []string
	logCtx, cancelLog := context.WithTimeout(ctx, 8*time.Second)
	defer cancelLog()
	reader, logErr := w.cli.ContainerLogs(logCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
		Timestamps: true,
	})
	if logErr == nil {
		data, readErr := io.ReadAll(reader)
		reader.Close()
		if readErr == nil {
			logLines = parseDockerLogs(data)
		}
	}

	// Fetch memory limit from inspect (best-effort).
	var memLimit uint64
	inspCtx, cancelInsp := context.WithTimeout(ctx, 3*time.Second)
	defer cancelInsp()
	if insp, err := w.cli.ContainerInspect(inspCtx, containerID); err == nil {
		if insp.HostConfig != nil && insp.HostConfig.Memory > 0 {
			memLimit = uint64(insp.HostConfig.Memory)
		}
	}

	cap := &CapturedExit{
		ContainerID:   containerID,
		ContainerName: name,
		ExitCode:      exitCode,
		OOMKilled:     oomKilled,
		ExitReason:    exitReason,
		LogLines:      logLines,
		MemoryLimit:   memLimit,
		CapturedAt:    time.Unix(0, msg.TimeNano),
	}

	w.mu.Lock()
	w.pending[containerID] = cap
	w.mu.Unlock()
}
