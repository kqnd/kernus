package models

import (
	"fmt"
	"strings"
	"time"
)

type Container struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Status        ContainerStatus   `json:"status"`
	State         string            `json:"state"`
	Created       time.Time         `json:"created"`
	Started       time.Time         `json:"started"`
	Finished      time.Time         `json:"finished"`
	Ports         []Port            `json:"ports"`
	Mounts        []Mount           `json:"mounts"`
	Networks      []Network         `json:"networks"`
	Labels        map[string]string `json:"labels"`
	Command       string            `json:"command"`
	Stats         *ContainerStats   `json:"stats"`
	Health        *ContainerHealth  `json:"health"`
	RestartPolicy RestartPolicy     `json:"restart_policy"`
	ExitCode      int               `json:"exit_code"`
}

type ContainerStats struct {
	CPU       ContainerCPU     `json:"cpu"`
	Memory    ContainerMemory  `json:"memory"`
	Network   ContainerNetwork `json:"network"`
	BlockIO   ContainerBlockIO `json:"block_io"`
	PIDs      int              `json:"pids"`
	Timestamp time.Time        `json:"timestamp"`
}

type ContainerCPU struct {
	Usage      float64 `json:"usage"`
	System     float64 `json:"system"`
	Cores      int     `json:"cores"`
	Throttling struct {
		Periods          int64 `json:"periods"`
		ThrottledPeriods int64 `json:"throttled_periods"`
		ThrottledTime    int64 `json:"throttled_time"`
	} `json:"throttling"`
}

type ContainerMemory struct {
	Usage     int64 `json:"usage"`
	Limit     int64 `json:"limit"`
	Cache     int64 `json:"cache"`
	RSS       int64 `json:"rss"`
	Swap      int64 `json:"swap"`
	SwapLimit int64 `json:"swap_limit"`
	MaxUsage  int64 `json:"max_usage"`
}

type ContainerNetwork struct {
	RxBytes   int64 `json:"rx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	RxErrors  int64 `json:"rx_errors"`
	RxDropped int64 `json:"rx_dropped"`
	TxBytes   int64 `json:"tx_bytes"`
	TxPackets int64 `json:"tx_packets"`
	TxErrors  int64 `json:"tx_errors"`
	TxDropped int64 `json:"tx_dropped"`
}

type ContainerBlockIO struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`
}

type ContainerHealth struct {
	Status        HealthStatus `json:"status"`
	FailingStreak int          `json:"failing_streak"`
	Log           []HealthLog  `json:"log"`
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusStarting  HealthStatus = "starting"
	HealthStatusNone      HealthStatus = "none"
)

func (h HealthStatus) Color() string {
	switch h {
	case HealthStatusHealthy:
		return "green"
	case HealthStatusUnhealthy:
		return "red"
	case HealthStatusStarting:
		return "yellow"
	default:
		return "gray"
	}
}

func (h HealthStatus) Icon() string {
	switch h {
	case HealthStatusHealthy:
		return "✓"
	case HealthStatusUnhealthy:
		return "✗"
	case HealthStatusStarting:
		return "⟳"
	default:
		return "?"
	}
}

type HealthLog struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	ExitCode int       `json:"exit_code"`
	Output   string    `json:"output"`
}

type RestartPolicy struct {
	Name              string `json:"name"`
	MaximumRetryCount int    `json:"maximum_retry_count"`
}

type ContainerStatus string

const (
	StatusRunning    ContainerStatus = "running"
	StatusExited     ContainerStatus = "exited"
	StatusPaused     ContainerStatus = "paused"
	StatusStopped    ContainerStatus = "stopped"
	StatusCreated    ContainerStatus = "created"
	StatusRestarting ContainerStatus = "restarting"
	StatusRemoving   ContainerStatus = "removing"
	StatusDead       ContainerStatus = "dead"
)

func (s ContainerStatus) Color() string {
	switch s {
	case StatusRunning:
		return "green"
	case StatusExited, StatusStopped, StatusDead:
		return "red"
	case StatusPaused:
		return "yellow"
	case StatusCreated:
		return "blue"
	case StatusRestarting:
		return "orange"
	case StatusRemoving:
		return "purple"
	default:
		return "white"
	}
}

func (s ContainerStatus) Icon() string {
	switch s {
	case StatusRunning:
		return "▶"
	case StatusExited, StatusStopped:
		return "■"
	case StatusPaused:
		return "⏸"
	case StatusCreated:
		return "⚪"
	case StatusRestarting:
		return "⟳"
	case StatusRemoving:
		return "🗑"
	case StatusDead:
		return "✗"
	default:
		return "?"
	}
}

type Port struct {
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port,omitempty"`
	Type        string `json:"type"`
	IP          string `json:"ip,omitempty"`
}

func (p Port) String() string {
	if p.PublicPort > 0 {
		return fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
	}
	return fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
}

type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	Type        string `json:"type"`
	RW          bool   `json:"rw"`
}

func (m Mount) String() string {
	mode := "ro"
	if m.RW {
		mode = "rw"
	}
	return fmt.Sprintf("%s:%s (%s)", m.Source, m.Destination, mode)
}

type Network struct {
	Name       string `json:"name"`
	NetworkID  string `json:"network_id"`
	IPAddress  string `json:"ip_address"`
	Gateway    string `json:"gateway"`
	MacAddress string `json:"mac_address"`
}

func (m ContainerMemory) Percentage() float64 {
	if m.Limit == 0 {
		return 0
	}
	return float64(m.Usage) / float64(m.Limit) * 100
}

func (m ContainerMemory) String() string {
	return fmt.Sprintf("%.1fMB / %.1fMB (%.1f%%)",
		float64(m.Usage)/1024/1024,
		float64(m.Limit)/1024/1024,
		m.Percentage())
}

func (m ContainerMemory) CacheString() string {
	return fmt.Sprintf("Cache: %.1fMB | RSS: %.1fMB",
		float64(m.Cache)/1024/1024,
		float64(m.RSS)/1024/1024)
}

func (c ContainerCPU) String() string {
	return fmt.Sprintf("%.2f%% (%d cores)", c.Usage, c.Cores)
}

func (c ContainerCPU) ThrottleString() string {
	if c.Throttling.ThrottledPeriods == 0 {
		return "No throttling"
	}
	return fmt.Sprintf("Throttled: %d/%d periods", c.Throttling.ThrottledPeriods, c.Throttling.Periods)
}

func (n ContainerNetwork) String() string {
	return fmt.Sprintf("↓ %.1fMB ↑ %.1fMB",
		float64(n.RxBytes)/1024/1024,
		float64(n.TxBytes)/1024/1024)
}

func (n ContainerNetwork) PacketsString() string {
	return fmt.Sprintf("↓ %d pkts ↑ %d pkts", n.RxPackets, n.TxPackets)
}

func (n ContainerNetwork) ErrorsString() string {
	if n.RxErrors == 0 && n.TxErrors == 0 {
		return "No errors"
	}
	return fmt.Sprintf("Errors: ↓ %d ↑ %d", n.RxErrors, n.TxErrors)
}

func (b ContainerBlockIO) String() string {
	return fmt.Sprintf("Read: %.1fMB | Write: %.1fMB",
		float64(b.ReadBytes)/1024/1024,
		float64(b.WriteBytes)/1024/1024)
}

func (b ContainerBlockIO) OpsString() string {
	return fmt.Sprintf("Read: %d ops | Write: %d ops", b.ReadOps, b.WriteOps)
}

func (c *Container) ShortID() string {
	if len(c.ID) > 12 {
		return c.ID[:12]
	}
	return c.ID
}

func (c *Container) ShortName() string {
	return strings.TrimPrefix(c.Name, "/")
}

func (c *Container) Age() time.Duration {
	return time.Since(c.Created)
}

func (c *Container) Uptime() time.Duration {
	if c.Status == StatusRunning && !c.Started.IsZero() {
		return time.Since(c.Started)
	}
	return 0
}

func (c *Container) FormatAge() string {
	age := c.Age()
	return formatDuration(age)
}

func (c *Container) FormatUptime() string {
	uptime := c.Uptime()
	if uptime == 0 {
		return "Not running"
	}
	return formatDuration(uptime)
}

func (c *Container) MainPort() string {
	if len(c.Ports) == 0 {
		return "No ports"
	}

	for _, port := range c.Ports {
		if port.PublicPort > 0 {
			return port.String()
		}
	}

	return c.Ports[0].String()
}

func (c *Container) AllPorts() string {
	if len(c.Ports) == 0 {
		return "No ports"
	}

	var ports []string
	for _, port := range c.Ports {
		ports = append(ports, port.String())
	}
	return strings.Join(ports, ", ")
}

func (c *Container) ShortPort() string {
	if len(c.Ports) == 0 {
		return "None"
	}

	for _, port := range c.Ports {
		if port.PublicPort > 0 {
			return fmt.Sprintf("%d", port.PublicPort)
		}
	}

	return fmt.Sprintf("%d", c.Ports[0].PrivatePort)
}

func (c *Container) ImageTag() string {
	parts := strings.Split(c.Image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "latest"
}

func (c *Container) ImageName() string {
	parts := strings.Split(c.Image, ":")
	return parts[0]
}

func (c *Container) IsHealthy() bool {
	if c.Health == nil {
		return true
	}
	return c.Health.Status == HealthStatusHealthy
}

func (c *Container) GetCPUUsage() float64 {
	if c.Stats != nil {
		return c.Stats.CPU.Usage
	}
	return 0
}

func (c *Container) GetMemoryUsage() int64 {
	if c.Stats != nil {
		return c.Stats.Memory.Usage
	}
	return 0
}

func (c *Container) GetMemoryLimit() int64 {
	if c.Stats != nil {
		return c.Stats.Memory.Limit
	}
	return 0
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func MockContainers() []Container {
	now := time.Now()
	return []Container{
		{
			ID:      "abc123456789",
			Name:    "/nginx-web",
			Image:   "nginx:latest",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-2 * time.Hour),
			Started: now.Add(-2 * time.Hour),
			Ports: []Port{
				{PrivatePort: 80, PublicPort: 8080, Type: "tcp", IP: "0.0.0.0"},
				{PrivatePort: 443, PublicPort: 8443, Type: "tcp", IP: "0.0.0.0"},
			},
			Command: "nginx -g 'daemon off;'",
			Stats: &ContainerStats{
				CPU: ContainerCPU{
					Usage:  5.2,
					System: 12.5,
					Cores:  4,
				},
				Memory: ContainerMemory{
					Usage: 50 * 1024 * 1024,
					Limit: 512 * 1024 * 1024,
					Cache: 20 * 1024 * 1024,
					RSS:   30 * 1024 * 1024,
				},
				Network: ContainerNetwork{
					RxBytes:   1024 * 1024 * 100,
					TxBytes:   1024 * 1024 * 200,
					RxPackets: 15000,
					TxPackets: 12000,
				},
				BlockIO: ContainerBlockIO{
					ReadBytes:  1024 * 1024 * 50,
					WriteBytes: 1024 * 1024 * 25,
					ReadOps:    1500,
					WriteOps:   800,
				},
				PIDs:      12,
				Timestamp: now,
			},
			Health: &ContainerHealth{
				Status:        HealthStatusHealthy,
				FailingStreak: 0,
			},
			RestartPolicy: RestartPolicy{
				Name:              "unless-stopped",
				MaximumRetryCount: 0,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "web",
				"environment":                "production",
			},
		},
		{
			ID:      "def456789012",
			Name:    "/postgres-db",
			Image:   "postgres:13",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-24 * time.Hour),
			Started: now.Add(-24 * time.Hour),
			Ports: []Port{
				{PrivatePort: 5432, PublicPort: 5432, Type: "tcp", IP: "127.0.0.1"},
			},
			Command: "docker-entrypoint.sh postgres",
			Stats: &ContainerStats{
				CPU: ContainerCPU{
					Usage:  15.7,
					System: 8.3,
					Cores:  2,
				},
				Memory: ContainerMemory{
					Usage: 200 * 1024 * 1024,
					Limit: 1024 * 1024 * 1024,
					Cache: 150 * 1024 * 1024,
					RSS:   50 * 1024 * 1024,
				},
				Network: ContainerNetwork{
					RxBytes:   1024 * 1024 * 50,
					TxBytes:   1024 * 1024 * 75,
					RxPackets: 8000,
					TxPackets: 9000,
				},
				BlockIO: ContainerBlockIO{
					ReadBytes:  1024 * 1024 * 500,
					WriteBytes: 1024 * 1024 * 300,
					ReadOps:    25000,
					WriteOps:   15000,
				},
				PIDs:      25,
				Timestamp: now,
			},
			Health: &ContainerHealth{
				Status:        HealthStatusHealthy,
				FailingStreak: 0,
			},
			RestartPolicy: RestartPolicy{
				Name:              "always",
				MaximumRetryCount: 0,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "database",
			},
		},
		{
			ID:       "ghi789012345",
			Name:     "/redis-cache",
			Image:    "redis:6-alpine",
			Status:   StatusExited,
			State:    "exited",
			Created:  now.Add(-6 * time.Hour),
			Started:  time.Time{},
			Finished: now.Add(-1 * time.Hour),
			Ports: []Port{
				{PrivatePort: 6379, Type: "tcp"},
			},
			Command:  "redis-server",
			ExitCode: 1,
			RestartPolicy: RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 3,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "cache",
			},
		},
		{
			ID:      "jkl012345678",
			Name:    "/app-worker",
			Image:   "myapp:latest",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-30 * time.Minute),
			Started: now.Add(-30 * time.Minute),
			Ports: []Port{
				{PrivatePort: 3000, PublicPort: 3000, Type: "tcp", IP: "0.0.0.0"},
			},
			Command: "npm start",
			Stats: &ContainerStats{
				CPU: ContainerCPU{
					Usage:  25.4,
					System: 15.2,
					Cores:  1,
					Throttling: struct {
						Periods          int64 `json:"periods"`
						ThrottledPeriods int64 `json:"throttled_periods"`
						ThrottledTime    int64 `json:"throttled_time"`
					}{
						Periods:          1000,
						ThrottledPeriods: 50,
						ThrottledTime:    5000000,
					},
				},
				Memory: ContainerMemory{
					Usage: 150 * 1024 * 1024,
					Limit: 512 * 1024 * 1024,
					Cache: 30 * 1024 * 1024,
					RSS:   120 * 1024 * 1024,
				},
				Network: ContainerNetwork{
					RxBytes:   1024 * 1024 * 20,
					TxBytes:   1024 * 1024 * 30,
					RxPackets: 5000,
					TxPackets: 6000,
				},
				BlockIO: ContainerBlockIO{
					ReadBytes:  1024 * 1024 * 10,
					WriteBytes: 1024 * 1024 * 5,
					ReadOps:    500,
					WriteOps:   200,
				},
				PIDs:      8,
				Timestamp: now,
			},
			Health: &ContainerHealth{
				Status:        HealthStatusUnhealthy,
				FailingStreak: 3,
			},
			RestartPolicy: RestartPolicy{
				Name:              "unless-stopped",
				MaximumRetryCount: 0,
			},
			Labels: map[string]string{
				"version":     "1.2.3",
				"environment": "development",
			},
		},
		{
			ID:      "mno345678901",
			Name:    "/monitoring-grafana",
			Image:   "grafana/grafana:latest",
			Status:  StatusPaused,
			State:   "paused",
			Created: now.Add(-3 * time.Hour),
			Started: now.Add(-3 * time.Hour),
			Ports: []Port{
				{PrivatePort: 3000, PublicPort: 3001, Type: "tcp", IP: "0.0.0.0"},
			},
			Command: "/run.sh",
			Stats: &ContainerStats{
				Memory: ContainerMemory{
					Usage: 80 * 1024 * 1024,
					Limit: 256 * 1024 * 1024,
					Cache: 40 * 1024 * 1024,
					RSS:   40 * 1024 * 1024,
				},
				PIDs:      15,
				Timestamp: now,
			},
			Health: &ContainerHealth{
				Status:        HealthStatusNone,
				FailingStreak: 0,
			},
			RestartPolicy: RestartPolicy{
				Name:              "no",
				MaximumRetryCount: 0,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "monitoring",
			},
		},
	}
}
