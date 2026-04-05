package agent

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Collector struct {
	cli             *client.Client
	hostMemoryTotal uint64
}

func NewCollector(dockerHost string) (*Collector, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if strings.TrimSpace(dockerHost) != "" {
		opts = append(opts, client.WithHost(strings.TrimSpace(dockerHost)))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("cannot connect to Docker daemon: %w", err)
	}

	var hostMemTotal uint64
	if info, err := cli.Info(ctx); err == nil && info.MemTotal > 0 {
		hostMemTotal = uint64(info.MemTotal)
	} else {
		hostMemTotal = readHostMemoryTotal()
	}

	return &Collector{cli: cli, hostMemoryTotal: hostMemTotal}, nil
}

func (c *Collector) Close() error {
	if c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

func (c *Collector) CountContainers(ctx context.Context) (int, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return 0, fmt.Errorf("cannot list containers: %w", err)
	}
	return len(containers), nil
}

func (c *Collector) Collect(ctx context.Context) ([]ContainerMetric, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("cannot list containers: %w", err)
	}

	metrics := make([]ContainerMetric, 0, len(containers))
	for _, ct := range containers {
		metric, mErr := c.collectContainerMetric(ctx, ct)
		if mErr != nil {
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (c *Collector) collectContainerMetric(ctx context.Context, ct types.Container) (ContainerMetric, error) {
	name := ""
	if len(ct.Names) > 0 {
		name = strings.TrimPrefix(ct.Names[0], "/")
	}

	inspect, err := c.cli.ContainerInspect(ctx, ct.ID)
	if err != nil {
		return ContainerMetric{}, err
	}

	timestamp := time.Now().UTC()
	status := strings.ToLower(strings.TrimSpace(inspect.State.Status))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(ct.State))
	}
	health := "none"
	if inspect.State.Health != nil {
		h := strings.TrimSpace(inspect.State.Health.Status)
		if h != "" {
			health = h
		}
	}

	restartCount := uint16(65535)
	if inspect.RestartCount >= 0 && inspect.RestartCount < 65535 {
		restartCount = uint16(inspect.RestartCount)
	}

	metric := ContainerMetric{
		ID:           limitString(ct.ID, 64),
		Name:         limitString(name, 255),
		Status:       status,
		Health:       health,
		RestartCount: restartCount,
		Timestamp:    timestamp,
	}

	// Capture exit info for non-running containers (exited, dead, stopped)
	if inspect.State != nil && !inspect.State.Running {
		metric.ExitCode = int32(inspect.State.ExitCode)
		metric.OOMKilled = inspect.State.OOMKilled
		metric.ExitReason = ClassifyExitReason(inspect.State.ExitCode, inspect.State.OOMKilled)
	}

	if inspect.State == nil || !inspect.State.Running {
		return metric, nil
	}

	statsResp, err := c.cli.ContainerStats(ctx, ct.ID, false)
	if err != nil {
		return metric, nil
	}
	defer statsResp.Body.Close()

	body, err := io.ReadAll(statsResp.Body)
	if err != nil {
		return metric, nil
	}

	var stats types.StatsJSON
	if err := json.Unmarshal(body, &stats); err != nil {
		return metric, nil
	}

	metric.CPUPercent = float32(calculateCPUPercent(stats))
	metric.MemoryUsed = calculateMemoryUsed(stats)
	metric.MemoryLimit = calculateMemoryLimit(stats, c.hostMemoryTotal)
	metric.NetworkRxBytes, metric.NetworkTxBytes = calculateNetworkIO(stats)

	return metric, nil
}

func calculateCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if systemDelta <= 0 || cpuDelta < 0 {
		return 0
	}

	numCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if numCPUs <= 0 {
		numCPUs = 1
	}

	pct := (cpuDelta / systemDelta) * numCPUs * 100.0
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func calculateMemoryUsed(stats types.StatsJSON) uint64 {
	cache := uint64(0)
	if v, ok := stats.MemoryStats.Stats["cache"]; ok {
		cache = v
	}
	if stats.MemoryStats.Usage <= cache {
		return stats.MemoryStats.Usage
	}
	return stats.MemoryStats.Usage - cache
}

func calculateMemoryLimit(stats types.StatsJSON, hostTotal uint64) uint64 {
	limit := stats.MemoryStats.Limit
	if hostTotal > 0 && limit == hostTotal {
		return 0
	}
	return limit
}

func calculateNetworkIO(stats types.StatsJSON) (rxBytes, txBytes uint64) {
	for _, netStats := range stats.Networks {
		rxBytes += netStats.RxBytes
		txBytes += netStats.TxBytes
	}
	return rxBytes, txBytes
}

func readHostMemoryTotal() uint64 {
	if runtime.GOOS != "linux" {
		return 0
	}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}

	return 0
}

func limitString(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func (c *Collector) GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error) {
	tail := fmt.Sprintf("%d", lines)
	reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get logs for container %s: %w", limitString(containerID, 12), err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot read logs: %w", err)
	}

	return parseDockerLogs(data), nil
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
