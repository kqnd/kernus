package agent

import (
	"context"
	"time"
)

type MetricCollector interface {
	Collect(ctx context.Context) ([]ContainerMetric, error)
	Close() error
}

type IngestRequest struct {
	HostName   string            `json:"host_name"`
	SentAt     time.Time         `json:"sent_at"`
	Containers []ContainerMetric `json:"containers"`
}

type ContainerMetric struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CPUPercent   float32   `json:"cpu_percent"`
	MemoryUsed   uint64    `json:"memory_used"`
	MemoryLimit  uint64    `json:"memory_limit"`
	RestartCount uint16    `json:"restart_count"`
	Status       string    `json:"status"`
	Health       string    `json:"health"`
	Timestamp    time.Time `json:"timestamp"`
}
