package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kiev/kernus/internal/models"
)

type DockerClient interface {
	ListContainers(ctx context.Context, onlyRunning bool) ([]models.Container, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RestartContainer(ctx context.Context, containerID string) error
	PauseContainer(ctx context.Context, containerID string) error
	UnpauseContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	GetContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error)
	GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error)
	Ping(ctx context.Context) error
	Close() error
}

type Client struct {
	cli *client.Client
}

func NewClient(host string) (DockerClient, error) {
	var opts []client.Opt
	opts = append(opts, client.FromEnv)
	opts = append(opts, client.WithAPIVersionNegotiation())

	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		closeErr := cli.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("Docker ping failed: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("cannot connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) ListContainers(ctx context.Context, onlyRunning bool) ([]models.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All: !onlyRunning,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list containers: %w", err)
	}

	result := make([]models.Container, len(containers))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, ct := range containers {
		wg.Add(1)
		go func(idx int, ct types.Container) {
			defer wg.Done()
			mc := convertContainer(ct)

			if mc.Status == models.StatusRunning {
				stats, statsErr := c.GetContainerStats(ctx, ct.ID)
				if statsErr == nil {
					mu.Lock()
					mc.Stats = stats
					mu.Unlock()
				}
			}

			logs, logsErr := c.GetContainerLogs(ctx, ct.ID, 100)
			if logsErr == nil {
				mu.Lock()
				mc.Logs = logs
				mu.Unlock()
			}

			inspect, inspectErr := c.cli.ContainerInspect(ctx, ct.ID)
			if inspectErr == nil {
				started, parseErr := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
				if parseErr == nil && !started.IsZero() {
					mu.Lock()
					mc.Started = started
					mu.Unlock()
				}
				finished, parseErr := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt)
				if parseErr == nil && !finished.IsZero() {
					mu.Lock()
					mc.Finished = finished
					mu.Unlock()
				}
				mu.Lock()
				mc.ExitCode = inspect.State.ExitCode
				mc.OOMKilled = inspect.State.OOMKilled
				if !inspect.State.Running {
					mc.ExitReason = classifyExitReason(inspect.State.ExitCode, inspect.State.OOMKilled)
				}
				if inspect.State.Health != nil {
					mc.Health.Status = models.HealthStatus(inspect.State.Health.Status)
					mc.Health.FailingStreak = inspect.State.Health.FailingStreak
				}
				if inspect.HostConfig != nil {
					mc.RestartPolicy = models.RestartPolicy{
						Name:              string(inspect.HostConfig.RestartPolicy.Name),
						MaximumRetryCount: inspect.HostConfig.RestartPolicy.MaximumRetryCount,
					}
				}
				mu.Unlock()
			}

			mu.Lock()
			result[idx] = mc
			mu.Unlock()
		}(i, ct)
	}

	wg.Wait()
	return result, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("cannot start container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("cannot stop container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerRestart(ctx, containerID, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("cannot restart container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) PauseContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("cannot pause container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) UnpauseContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("cannot unpause container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
	if err != nil {
		return fmt.Errorf("cannot remove container %s: %w", shortID(containerID), err)
	}
	return nil
}

func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	resp, err := c.cli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("cannot get stats for %s: %w", shortID(containerID), err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read stats: %w", err)
	}

	return parseStats(data)
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error) {
	tail := fmt.Sprintf("%d", lines)
	reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get logs for %s: %w", shortID(containerID), err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot read logs: %w", err)
	}

	return parseDockerLogs(data), nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker ping failed: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func convertContainer(ct types.Container) models.Container {
	name := ""
	if len(ct.Names) > 0 {
		name = strings.TrimPrefix(ct.Names[0], "/")
	}

	status := detectStatus(ct.State)
	health := detectHealth(ct.Status)

	var ports []models.Port
	for _, p := range ct.Ports {
		ports = append(ports, models.Port{
			PrivatePort: int(p.PrivatePort),
			PublicPort:  int(p.PublicPort),
			Type:        p.Type,
			IP:          p.IP,
		})
	}

	var mounts []models.Mount
	for _, m := range ct.Mounts {
		mounts = append(mounts, models.Mount{
			Source:      m.Source,
			Destination: m.Destination,
			Type:        string(m.Type),
			Mode:        m.Mode,
		})
	}

	var networks []models.Network
	if ct.NetworkSettings != nil {
		for netName, net := range ct.NetworkSettings.Networks {
			networks = append(networks, models.Network{
				Name:    netName,
				IP:      net.IPAddress,
				Gateway: net.Gateway,
				MAC:     net.MacAddress,
			})
		}
	}

	return models.Container{
		ID:       ct.ID,
		Name:     name,
		Image:    ct.Image,
		Status:   status,
		State:    ct.State,
		Created:  time.Unix(ct.Created, 0),
		Started:  time.Unix(ct.Created, 0),
		Ports:    ports,
		Mounts:   mounts,
		Networks: networks,
		Labels:   ct.Labels,
		Command:  ct.Command,
		Health: models.ContainerHealth{
			Status: health,
		},
	}
}

func detectStatus(state string) models.ContainerStatus {
	switch strings.ToLower(state) {
	case "running":
		return models.StatusRunning
	case "exited":
		return models.StatusExited
	case "paused":
		return models.StatusPaused
	case "created":
		return models.StatusCreated
	case "restarting":
		return models.StatusRestarting
	case "removing":
		return models.StatusRemoving
	case "dead":
		return models.StatusDead
	default:
		return models.StatusStopped
	}
}

func classifyExitReason(exitCode int, oomKilled bool) string {
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

func detectHealth(statusStr string) models.HealthStatus {
	lower := strings.ToLower(statusStr)
	if strings.Contains(lower, "(healthy)") {
		return models.HealthHealthy
	}
	if strings.Contains(lower, "(unhealthy)") {
		return models.HealthUnhealthy
	}
	if strings.Contains(lower, "(health: starting)") {
		return models.HealthStarting
	}
	return models.HealthNone
}

func parseDockerLogs(data []byte) []string {
	var lines []string
	offset := 0

	for offset < len(data) {
		if offset+8 <= len(data) && (data[offset] == 1 || data[offset] == 2) && data[offset+1] == 0 && data[offset+2] == 0 && data[offset+3] == 0 {
			size := binary.BigEndian.Uint32(data[offset+4 : offset+8])
			offset += 8
			if offset+int(size) <= len(data) {
				line := strings.TrimRight(string(data[offset:offset+int(size)]), "\n\r")
				if line != "" {
					lines = append(lines, line)
				}
				offset += int(size)
				continue
			}
		}

		end := offset
		for end < len(data) && data[end] != '\n' {
			end++
		}
		line := strings.TrimRight(string(data[offset:end]), "\r")
		if line != "" {
			lines = append(lines, line)
		}
		offset = end + 1
	}

	return lines
}

type rawStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
		ThrottlingData struct {
			ThrottledTime    uint64 `json:"throttled_time"`
			Periods          uint64 `json:"periods"`
			ThrottledPeriods uint64 `json:"throttled_periods"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage    uint64 `json:"usage"`
		Limit    uint64 `json:"limit"`
		MaxUsage uint64 `json:"max_usage"`
		Stats    struct {
			Cache        uint64 `json:"cache"`
			RSS          uint64 `json:"rss"`
			Swap         uint64 `json:"swap"`
			InactiveFile uint64 `json:"inactive_file"`
		} `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes   uint64 `json:"rx_bytes"`
		RxPackets uint64 `json:"rx_packets"`
		RxErrors  uint64 `json:"rx_errors"`
		RxDropped uint64 `json:"rx_dropped"`
		TxBytes   uint64 `json:"tx_bytes"`
		TxPackets uint64 `json:"tx_packets"`
		TxErrors  uint64 `json:"tx_errors"`
		TxDropped uint64 `json:"tx_dropped"`
	} `json:"networks"`
	BlkioStats struct {
		IOServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
		IOServicedRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_serviced_recursive"`
	} `json:"blkio_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

func parseStats(data []byte) (*models.ContainerStats, error) {
	var raw rawStats
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("cannot parse stats JSON: %w", err)
	}

	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	numCPUs := raw.CPUStats.OnlineCPUs
	if numCPUs == 0 {
		numCPUs = 1
	}

	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(numCPUs) * 100.0
	}

	throttlePercent := 0.0
	if raw.CPUStats.ThrottlingData.Periods > 0 {
		throttlePercent = float64(raw.CPUStats.ThrottlingData.ThrottledPeriods) / float64(raw.CPUStats.ThrottlingData.Periods) * 100.0
	}

	cache := raw.MemoryStats.Stats.Cache
	if cache == 0 {
		cache = raw.MemoryStats.Stats.InactiveFile
	}
	memUsage := raw.MemoryStats.Usage - cache

	var net models.ContainerNetwork
	for _, n := range raw.Networks {
		net.RxBytes += int64(n.RxBytes)
		net.RxPackets += int64(n.RxPackets)
		net.RxErrors += int64(n.RxErrors)
		net.RxDropped += int64(n.RxDropped)
		net.TxBytes += int64(n.TxBytes)
		net.TxPackets += int64(n.TxPackets)
		net.TxErrors += int64(n.TxErrors)
		net.TxDropped += int64(n.TxDropped)
	}

	var blkRead, blkWrite int64
	var blkReadOps, blkWriteOps int64
	for _, entry := range raw.BlkioStats.IOServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			blkRead += int64(entry.Value)
		case "write":
			blkWrite += int64(entry.Value)
		}
	}
	for _, entry := range raw.BlkioStats.IOServicedRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			blkReadOps += int64(entry.Value)
		case "write":
			blkWriteOps += int64(entry.Value)
		}
	}

	return &models.ContainerStats{
		CPU: models.ContainerCPU{
			Usage:      cpuPercent,
			System:     float64(raw.CPUStats.SystemCPUUsage),
			Cores:      numCPUs,
			Throttling: throttlePercent,
		},
		Memory: models.ContainerMemory{
			Usage:    int64(memUsage),
			Limit:    int64(raw.MemoryStats.Limit),
			Cache:    int64(cache),
			RSS:      int64(raw.MemoryStats.Stats.RSS),
			Swap:     int64(raw.MemoryStats.Stats.Swap),
			MaxUsage: int64(raw.MemoryStats.MaxUsage),
		},
		Network: net,
		BlockIO: models.ContainerBlockIO{
			ReadBytes:  blkRead,
			WriteBytes: blkWrite,
			ReadOps:    blkReadOps,
			WriteOps:   blkWriteOps,
		},
		PIDs:      int(raw.PidsStats.Current),
		Timestamp: time.Now(),
	}, nil
}
