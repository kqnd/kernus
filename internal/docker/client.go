package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kqnd/kernus/internal/models"
)

type Client struct {
	cli *client.Client
	ctx context.Context
}

type dockerStats struct {
	Read    time.Time `json:"read"`
	Preread time.Time `json:"preread"`
	PIDs    struct {
		Current int `json:"current"`
	} `json:"pids_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_service_bytes_recursive"`
		IoServicedRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int64  `json:"value"`
		} `json:"io_serviced_recursive"`
	} `json:"blkio_stats"`
	NumProcs     int `json:"num_procs"`
	StorageStats struct {
	} `json:"storage_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int64 `json:"periods"`
			ThrottledPeriods int64 `json:"throttled_periods"`
			ThrottledTime    int64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage    int64 `json:"usage"`
		MaxUsage int64 `json:"max_usage"`
		Stats    struct {
			ActiveAnon              int64 `json:"active_anon"`
			ActiveFile              int64 `json:"active_file"`
			Cache                   int64 `json:"cache"`
			Dirty                   int64 `json:"dirty"`
			HierarchicalMemoryLimit int64 `json:"hierarchical_memory_limit"`
			HierarchicalMemswLimit  int64 `json:"hierarchical_memsw_limit"`
			InactiveAnon            int64 `json:"inactive_anon"`
			InactiveFile            int64 `json:"inactive_file"`
			MappedFile              int64 `json:"mapped_file"`
			Pgfault                 int64 `json:"pgfault"`
			Pgmajfault              int64 `json:"pgmajfault"`
			Pgpgin                  int64 `json:"pgpgin"`
			Pgpgout                 int64 `json:"pgpgout"`
			RSS                     int64 `json:"rss"`
			RSSHuge                 int64 `json:"rss_huge"`
			TotalActiveAnon         int64 `json:"total_active_anon"`
			TotalActiveFile         int64 `json:"total_active_file"`
			TotalCache              int64 `json:"total_cache"`
			TotalDirty              int64 `json:"total_dirty"`
			TotalInactiveAnon       int64 `json:"total_inactive_anon"`
			TotalInactiveFile       int64 `json:"total_inactive_file"`
			TotalMappedFile         int64 `json:"total_mapped_file"`
			TotalPgfault            int64 `json:"total_pgfault"`
			TotalPgmajfault         int64 `json:"total_pgmajfault"`
			TotalPgpgin             int64 `json:"total_pgpgin"`
			TotalPgpgout            int64 `json:"total_pgpgout"`
			TotalRSS                int64 `json:"total_rss"`
			TotalRSSHuge            int64 `json:"total_rss_huge"`
			TotalUnevictable        int64 `json:"total_unevictable"`
			TotalWriteback          int64 `json:"total_writeback"`
			Unevictable             int64 `json:"unevictable"`
			Writeback               int64 `json:"writeback"`
		} `json:"stats"`
		Limit int64 `json:"limit"`
	} `json:"memory_stats"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	Networks map[string]struct {
		RxBytes   int64 `json:"rx_bytes"`
		RxPackets int64 `json:"rx_packets"`
		RxErrors  int64 `json:"rx_errors"`
		RxDropped int64 `json:"rx_dropped"`
		TxBytes   int64 `json:"tx_bytes"`
		TxPackets int64 `json:"tx_packets"`
		TxErrors  int64 `json:"tx_errors"`
		TxDropped int64 `json:"tx_dropped"`
	} `json:"networks"`
}

func NewClient(host string) (*Client, error) {
	var opts []client.Opt
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}
func (c *Client) ListContainers(onlyRunning bool) ([]models.Container, error) {
	options := container.ListOptions{
		All: !onlyRunning,
	}

	containers, err := c.cli.ContainerList(c.ctx, options)
	if err != nil {
		return nil, err
	}

	result := make([]models.Container, 0, len(containers))
	for _, container := range containers {
		modelContainer := c.convertContainer(container)

		if modelContainer.Status == models.StatusRunning {
			if stats, err := c.GetContainerStats(container.ID); err == nil {
				modelContainer.Stats = stats
			}
		}

		if logs, err := c.GetContainerLogs(container.ID, 100); err == nil {
			modelContainer.Logs = logs
		} else {
			modelContainer.Logs = make([]string, 0)
		}

		result = append(result, modelContainer)
	}
	return result, nil
}

func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(containerID string) error {
	timeoutSecs := 30
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	})
}

func (c *Client) RestartContainer(containerID string) error {
	timeoutSecs := 30
	return c.cli.ContainerRestart(c.ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	})
}

func (c *Client) PauseContainer(containerID string) error {
	return c.cli.ContainerPause(c.ctx, containerID)
}

func (c *Client) UnpauseContainer(containerID string) error {
	return c.cli.ContainerUnpause(c.ctx, containerID)
}

func (c *Client) RemoveContainer(containerID string, force bool) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

func (c *Client) GetContainerStats(containerID string) (*models.ContainerStats, error) {
	stats, err := c.cli.ContainerStats(c.ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var dockerStat dockerStats
	if err := json.NewDecoder(stats.Body).Decode(&dockerStat); err != nil {
		return nil, err
	}

	return c.convertStats(&dockerStat), nil
}

func (c *Client) GetContainerLogs(containerID string, lines int) ([]string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Details:    false,
	}

	if lines > 0 {
		tailStr := fmt.Sprintf("%d", lines)
		options.Tail = tailStr
	}

	logs, err := c.cli.ContainerLogs(c.ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	content, err := io.ReadAll(logs)
	if err != nil {
		return nil, err
	}

	return c.parseDockerLogs(content), nil
}

func (c *Client) parseDockerLogs(content []byte) []string {
	logLines := make([]string, 0)

	i := 0
	for i < len(content) {
		if i+8 > len(content) {
			break
		}

		msgSize := int(content[i+4])<<24 | int(content[i+5])<<16 | int(content[i+6])<<8 | int(content[i+7])

		if i+8+msgSize > len(content) {
			break
		}

		message := string(content[i+8 : i+8+msgSize])

		lines := strings.Split(strings.TrimRight(message, "\n\r"), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				logLines = append(logLines, trimmed)
			}
		}

		i += 8 + msgSize
	}

	if len(logLines) == 0 {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if len(trimmed) > 8 && trimmed[0] < 32 {
					logLines = append(logLines, trimmed[8:])
				} else {
					logLines = append(logLines, trimmed)
				}
			}
		}
	}

	return logLines
}

func (c *Client) RefreshContainerLogs(containerID string, lines int) ([]string, error) {
	return c.GetContainerLogs(containerID, lines)
}

func (c *Client) InspectContainer(containerID string) (*types.ContainerJSON, error) {
	resp, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) convertStats(stats *dockerStats) *models.ContainerStats {
	cpuUsage := c.calculateCPUPercentage(stats)

	var totalRx, totalTx, totalRxPackets, totalTxPackets, totalRxErrors, totalTxErrors, totalRxDropped, totalTxDropped int64
	for _, netStats := range stats.Networks {
		totalRx += netStats.RxBytes
		totalTx += netStats.TxBytes
		totalRxPackets += netStats.RxPackets
		totalTxPackets += netStats.TxPackets
		totalRxErrors += netStats.RxErrors
		totalTxErrors += netStats.TxErrors
		totalRxDropped += netStats.RxDropped
		totalTxDropped += netStats.TxDropped
	}

	var readBytes, writeBytes, readOps, writeOps int64
	for _, blkio := range stats.BlkioStats.IoServiceBytesRecursive {
		if blkio.Op == "Read" {
			readBytes += blkio.Value
		} else if blkio.Op == "Write" {
			writeBytes += blkio.Value
		}
	}
	for _, blkio := range stats.BlkioStats.IoServicedRecursive {
		if blkio.Op == "Read" {
			readOps += blkio.Value
		} else if blkio.Op == "Write" {
			writeOps += blkio.Value
		}
	}

	return &models.ContainerStats{
		CPU: models.ContainerCPU{
			Usage:  cpuUsage,
			System: c.calculateSystemCPUPercentage(stats),
			Cores:  stats.CPUStats.OnlineCpus,
			Throttling: struct {
				Periods          int64 `json:"periods"`
				ThrottledPeriods int64 `json:"throttled_periods"`
				ThrottledTime    int64 `json:"throttled_time"`
			}{
				Periods:          stats.CPUStats.ThrottlingData.Periods,
				ThrottledPeriods: stats.CPUStats.ThrottlingData.ThrottledPeriods,
				ThrottledTime:    stats.CPUStats.ThrottlingData.ThrottledTime,
			},
		},
		Memory: models.ContainerMemory{
			Usage:    stats.MemoryStats.Usage,
			Limit:    stats.MemoryStats.Limit,
			Cache:    stats.MemoryStats.Stats.Cache,
			RSS:      stats.MemoryStats.Stats.RSS,
			MaxUsage: stats.MemoryStats.MaxUsage,
		},
		Network: models.ContainerNetwork{
			RxBytes:   totalRx,
			RxPackets: totalRxPackets,
			RxErrors:  totalRxErrors,
			RxDropped: totalRxDropped,
			TxBytes:   totalTx,
			TxPackets: totalTxPackets,
			TxErrors:  totalTxErrors,
			TxDropped: totalTxDropped,
		},
		BlockIO: models.ContainerBlockIO{
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
			ReadOps:    readOps,
			WriteOps:   writeOps,
		},
		PIDs:      stats.PIDs.Current,
		Timestamp: stats.Read,
	}
}

func (c *Client) calculateCPUPercentage(stats *dockerStats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCpus) * 100.0
	}
	return 0.0
}

func (c *Client) calculateSystemCPUPercentage(stats *dockerStats) float64 {
	return float64(stats.CPUStats.SystemCPUUsage) / 1e9 * 100.0
}

func (c *Client) convertContainer(container types.Container) models.Container {
	name := container.Names[0]
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}

	ports := make([]models.Port, 0, len(container.Ports))
	for _, port := range container.Ports {
		modelPort := models.Port{
			PrivatePort: int(port.PrivatePort),
			PublicPort:  int(port.PublicPort),
			Type:        port.Type,
			IP:          port.IP,
		}
		ports = append(ports, modelPort)
	}

	networks := make([]models.Network, 0, len(container.NetworkSettings.Networks))
	for name, network := range container.NetworkSettings.Networks {
		modelNetwork := models.Network{
			Name:       name,
			NetworkID:  network.NetworkID,
			IPAddress:  network.IPAddress,
			Gateway:    network.Gateway,
			MacAddress: network.MacAddress,
		}
		networks = append(networks, modelNetwork)
	}

	mounts := make([]models.Mount, 0, len(container.Mounts))
	for _, mount := range container.Mounts {
		modelMount := models.Mount{
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			Type:        string(mount.Type),
			RW:          mount.RW,
		}
		mounts = append(mounts, modelMount)
	}

	var restartPolicy models.RestartPolicy

	var health *models.ContainerHealth
	if container.State == "running" {
		healthStatus := models.HealthStatusNone
		if strings.Contains(container.Status, "healthy") {
			healthStatus = models.HealthStatusHealthy
		} else if strings.Contains(container.Status, "unhealthy") {
			healthStatus = models.HealthStatusUnhealthy
		} else if strings.Contains(container.Status, "starting") {
			healthStatus = models.HealthStatusStarting
		}

		health = &models.ContainerHealth{
			Status:        healthStatus,
			FailingStreak: 0,
		}
	}

	return models.Container{
		ID:            container.ID,
		Name:          name,
		Image:         container.Image,
		Status:        models.ContainerStatus(container.State),
		State:         container.Status,
		Created:       time.Unix(container.Created, 0),
		Started:       time.Unix(container.Created, 0),
		Ports:         ports,
		Networks:      networks,
		Mounts:        mounts,
		Labels:        container.Labels,
		Command:       container.Command,
		RestartPolicy: restartPolicy,
		Health:        health,
	}
}

func (c *Client) Ping() error {
	_, err := c.cli.Ping(c.ctx)
	return err
}
