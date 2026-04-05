package agent

import (
	"context"
	"time"
)

type MetricCollector interface {
	Collect(ctx context.Context) ([]ContainerMetric, error)
	GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error)
	Close() error
}

type IngestRequest struct {
	HostName   string            `json:"host_name"`
	SentAt     time.Time         `json:"sent_at"`
	Containers []ContainerMetric `json:"containers"`
}

type LogSnapshotRequest struct {
	HostName  string        `json:"host_name"`
	SentAt    time.Time     `json:"sent_at"`
	Snapshots []LogSnapshot `json:"snapshots"`
}

type LogSnapshot struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Timestamp     time.Time `json:"timestamp"`
	EventType     string    `json:"event_type"`
	ExitCode      int32     `json:"exit_code"`
	ExitReason    string    `json:"exit_reason"`
	LogLines      []string  `json:"log_lines"`
	CPUPercent    float32   `json:"cpu_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryLimit   uint64    `json:"memory_limit"`
}

type ContainerMetric struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CPUPercent     float32   `json:"cpu_percent"`
	MemoryUsed     uint64    `json:"memory_used"`
	MemoryLimit    uint64    `json:"memory_limit"`
	RestartCount   uint16    `json:"restart_count"`
	Status         string    `json:"status"`
	Health         string    `json:"health"`
	ExitCode       int32     `json:"exit_code"`
	ExitReason     string    `json:"exit_reason"`
	OOMKilled      bool      `json:"oom_killed"`
	NetworkRxBytes uint64    `json:"network_rx_bytes"`
	NetworkTxBytes uint64    `json:"network_tx_bytes"`
	Timestamp      time.Time `json:"timestamp"`
}

// ClassifyExitReason determines a human-readable exit reason from container state.
func ClassifyExitReason(exitCode int, oomKilled bool) string {
	switch {
	case exitCode == 0:
		return "clean_stop"
	case oomKilled:
		return "oom_killed"
	case exitCode == 137 && !oomKilled:
		return "force_killed"
	case exitCode >= 1 && exitCode <= 126:
		return "app_crashed"
	default:
		return "unknown"
	}
}
