package models

import (
	"fmt"
	"strings"
	"time"
)

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
	case StatusPaused:
		return "yellow"
	case StatusRestarting:
		return "yellow"
	case StatusExited, StatusDead, StatusStopped:
		return "red"
	case StatusCreated:
		return "blue"
	case StatusRemoving:
		return "gray"
	default:
		return "white"
	}
}

func (s ContainerStatus) Icon() string {
	switch s {
	case StatusRunning:
		return "▶"
	case StatusPaused:
		return "⏸"
	case StatusExited, StatusDead, StatusStopped:
		return "■"
	case StatusCreated:
		return "○"
	case StatusRestarting:
		return "↻"
	case StatusRemoving:
		return "✕"
	default:
		return "?"
	}
}

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthStarting  HealthStatus = "starting"
	HealthNone      HealthStatus = "none"
)

func (h HealthStatus) Color() string {
	switch h {
	case HealthHealthy:
		return "green"
	case HealthUnhealthy:
		return "red"
	case HealthStarting:
		return "yellow"
	default:
		return "gray"
	}
}

func (h HealthStatus) Icon() string {
	switch h {
	case HealthHealthy:
		return "✓"
	case HealthUnhealthy:
		return "✗"
	case HealthStarting:
		return "…"
	default:
		return "-"
	}
}

type Container struct {
	ID            string
	Name          string
	Image         string
	Status        ContainerStatus
	State         string
	Created       time.Time
	Started       time.Time
	Finished      time.Time
	Ports         []Port
	Mounts        []Mount
	Networks      []Network
	Labels        map[string]string
	Command       string
	Stats         *ContainerStats
	Health        ContainerHealth
	RestartPolicy RestartPolicy
	ExitCode      int
	ExitReason    string
	OOMKilled     bool
	Logs          []string
}

type ContainerStats struct {
	CPU       ContainerCPU
	Memory    ContainerMemory
	Network   ContainerNetwork
	BlockIO   ContainerBlockIO
	PIDs      int
	Timestamp time.Time
}

type ContainerCPU struct {
	Usage      float64
	System     float64
	Cores      int
	Throttling float64
}

type ContainerMemory struct {
	Usage    int64
	Limit    int64
	Cache    int64
	RSS      int64
	Swap     int64
	MaxUsage int64
}

type ContainerNetwork struct {
	RxBytes   int64
	RxPackets int64
	RxErrors  int64
	RxDropped int64
	TxBytes   int64
	TxPackets int64
	TxErrors  int64
	TxDropped int64
}

type ContainerBlockIO struct {
	ReadBytes  int64
	WriteBytes int64
	ReadOps    int64
	WriteOps   int64
}

type ContainerHealth struct {
	Status        HealthStatus
	FailingStreak int
	Log           []string
}

type Port struct {
	PrivatePort int
	PublicPort  int
	Type        string
	IP          string
}

type Mount struct {
	Source      string
	Destination string
	Type        string
	Mode        string
}

type Network struct {
	Name    string
	IP      string
	Gateway string
	MAC     string
}

type RestartPolicy struct {
	Name              string
	MaximumRetryCount int
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

func (c *Container) FormatAge() string {
	return formatDuration(c.Age())
}

func (c *Container) Uptime() time.Duration {
	if c.Started.IsZero() {
		return 0
	}
	if c.Status != StatusRunning {
		if !c.Finished.IsZero() {
			return c.Finished.Sub(c.Started)
		}
		return 0
	}
	return time.Since(c.Started)
}

func (c *Container) FormatUptime() string {
	up := c.Uptime()
	if up == 0 {
		return "N/A"
	}
	return formatDuration(up)
}

func (c *Container) MainPort() string {
	if len(c.Ports) == 0 {
		return ""
	}
	p := c.Ports[0]
	if p.PublicPort > 0 {
		return fmt.Sprintf("%d", p.PublicPort)
	}
	return fmt.Sprintf("%d", p.PrivatePort)
}

func (c *Container) AllPorts() string {
	if len(c.Ports) == 0 {
		return "N/A"
	}
	var parts []string
	for _, p := range c.Ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

func (c *Container) ShortPort() string {
	if len(c.Ports) == 0 {
		return ""
	}
	p := c.Ports[0]
	if p.PublicPort > 0 {
		return fmt.Sprintf("%d→%d", p.PublicPort, p.PrivatePort)
	}
	return fmt.Sprintf("%d", p.PrivatePort)
}

func (c *Container) ImageName() string {
	parts := strings.Split(c.Image, ":")
	return parts[0]
}

func (c *Container) ImageTag() string {
	parts := strings.Split(c.Image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "latest"
}

func (c *Container) IsHealthy() bool {
	return c.Health.Status == HealthHealthy
}

func (c *Container) GetCPUUsage() float64 {
	if c.Stats == nil {
		return 0
	}
	return c.Stats.CPU.Usage
}

func (c *Container) GetMemoryUsage() int64 {
	if c.Stats == nil {
		return 0
	}
	return c.Stats.Memory.Usage
}

func (c *Container) GetMemoryLimit() int64 {
	if c.Stats == nil {
		return 0
	}
	return c.Stats.Memory.Limit
}

func (c *Container) GetRecentLogs(maxLines int) []string {
	if len(c.Logs) <= maxLines {
		return c.Logs
	}
	return c.Logs[len(c.Logs)-maxLines:]
}

func (m *ContainerMemory) Percentage() float64 {
	if m.Limit == 0 {
		return 0
	}
	return float64(m.Usage) / float64(m.Limit) * 100
}

func (m *ContainerMemory) String() string {
	return fmt.Sprintf("%s / %s (%.1f%%)",
		FormatBytesHuman(m.Usage),
		FormatBytesHuman(m.Limit),
		m.Percentage(),
	)
}

func (n *ContainerNetwork) String() string {
	return fmt.Sprintf("↓ %s ↑ %s",
		FormatBytesHuman(n.RxBytes),
		FormatBytesHuman(n.TxBytes),
	)
}

func (b *ContainerBlockIO) String() string {
	return fmt.Sprintf("R: %s W: %s",
		FormatBytesHuman(b.ReadBytes),
		FormatBytesHuman(b.WriteBytes),
	)
}

func (c *ContainerCPU) ThrottleString() string {
	if c.Throttling == 0 {
		return "none"
	}
	return fmt.Sprintf("%.1f%%", c.Throttling)
}

func FormatBytesHuman(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1fTB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

func MockContainers() []Container {
	now := time.Now()
	return []Container{
		{
			ID:      "a1b2c3d4ef5678901234567890abcdef",
			Name:    "nginx-web-1",
			Image:   "nginx:latest",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-48 * time.Hour),
			Started: now.Add(-2 * time.Hour),
			Ports: []Port{
				{PrivatePort: 80, PublicPort: 8080, Type: "tcp", IP: "0.0.0.0"},
			},
			Mounts: []Mount{
				{Source: "/var/www/html", Destination: "/usr/share/nginx/html", Type: "bind", Mode: "rw"},
			},
			Networks: []Network{
				{Name: "bridge", IP: "172.17.0.2", Gateway: "172.17.0.1", MAC: "02:42:ac:11:00:02"},
			},
			Labels:  map[string]string{"com.docker.compose.service": "web", "version": "1.0"},
			Command: "nginx -g 'daemon off;'",
			Stats: &ContainerStats{
				CPU:     ContainerCPU{Usage: 5.2, System: 2.1, Cores: 4, Throttling: 0},
				Memory:  ContainerMemory{Usage: 52428800, Limit: 536870912, Cache: 10485760, RSS: 41943040, Swap: 0, MaxUsage: 67108864},
				Network: ContainerNetwork{RxBytes: 104857600, RxPackets: 50000, RxErrors: 0, RxDropped: 0, TxBytes: 209715200, TxPackets: 75000, TxErrors: 0, TxDropped: 0},
				BlockIO: ContainerBlockIO{ReadBytes: 52428800, WriteBytes: 26214400, ReadOps: 1000, WriteOps: 500},
				PIDs:    5,
			},
			Health:        ContainerHealth{Status: HealthHealthy, FailingStreak: 0},
			RestartPolicy: RestartPolicy{Name: "always", MaximumRetryCount: 0},
			ExitCode:      0,
			Logs:          []string{"2026-03-23T13:00:00.000Z [info] Starting nginx...", "2026-03-23T13:00:01.000Z [info] nginx/1.25.3", "2026-03-23T13:00:01.500Z [info] Listening on port 80"},
		},
		{
			ID:      "e5f6g7h8ij9012345678901234abcdef",
			Name:    "nginx-web-2",
			Image:   "nginx:latest",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-48 * time.Hour),
			Started: now.Add(-1 * time.Hour),
			Ports: []Port{
				{PrivatePort: 80, PublicPort: 8081, Type: "tcp", IP: "0.0.0.0"},
			},
			Networks: []Network{
				{Name: "bridge", IP: "172.17.0.3", Gateway: "172.17.0.1", MAC: "02:42:ac:11:00:03"},
			},
			Labels:  map[string]string{"com.docker.compose.service": "web", "version": "1.0"},
			Command: "nginx -g 'daemon off;'",
			Stats: &ContainerStats{
				CPU:     ContainerCPU{Usage: 3.1, System: 1.5, Cores: 4, Throttling: 0},
				Memory:  ContainerMemory{Usage: 41943040, Limit: 536870912, Cache: 8388608, RSS: 33554432, Swap: 0, MaxUsage: 58720256},
				Network: ContainerNetwork{RxBytes: 83886080, RxPackets: 40000, TxBytes: 167772160, TxPackets: 60000},
				BlockIO: ContainerBlockIO{ReadBytes: 41943040, WriteBytes: 20971520, ReadOps: 800, WriteOps: 400},
				PIDs:    4,
			},
			Health:        ContainerHealth{Status: HealthHealthy, FailingStreak: 0},
			RestartPolicy: RestartPolicy{Name: "always", MaximumRetryCount: 0},
			ExitCode:      0,
			Logs:          []string{"2026-03-23T14:00:00.000Z [info] Starting nginx...", "2026-03-23T14:00:01.000Z [info] Listening on port 80"},
		},
		{
			ID:       "i9j0k1l2mn3456789012345678abcdef",
			Name:     "nginx-web-3",
			Image:    "nginx:1.24",
			Status:   StatusExited,
			State:    "exited",
			Created:  now.Add(-72 * time.Hour),
			Started:  now.Add(-24 * time.Hour),
			Finished: now.Add(-2 * time.Hour),
			Ports: []Port{
				{PrivatePort: 80, PublicPort: 8082, Type: "tcp", IP: "0.0.0.0"},
			},
			Networks: []Network{
				{Name: "bridge", IP: "172.17.0.4", Gateway: "172.17.0.1", MAC: "02:42:ac:11:00:04"},
			},
			Labels:        map[string]string{"com.docker.compose.service": "web"},
			Command:       "nginx -g 'daemon off;'",
			Health:        ContainerHealth{Status: HealthNone},
			RestartPolicy: RestartPolicy{Name: "no", MaximumRetryCount: 0},
			ExitCode:      137,
			ExitReason:    "oom_killed",
			OOMKilled:     true,
			Logs:          []string{"2026-03-22T13:00:00.000Z [info] Starting nginx...", "2026-03-22T13:00:01.000Z [error] Failed to bind port 80", "2026-03-22T13:00:01.500Z [fatal] Exiting"},
		},
		{
			ID:      "abc12345de6789012345678901abcdef",
			Name:    "postgres-db",
			Image:   "postgres:16",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-168 * time.Hour),
			Started: now.Add(-72 * time.Hour),
			Ports: []Port{
				{PrivatePort: 5432, PublicPort: 5432, Type: "tcp", IP: "0.0.0.0"},
			},
			Mounts: []Mount{
				{Source: "/var/lib/postgresql/data", Destination: "/var/lib/postgresql/data", Type: "volume", Mode: "rw"},
			},
			Networks: []Network{
				{Name: "backend", IP: "172.18.0.2", Gateway: "172.18.0.1", MAC: "02:42:ac:12:00:02"},
			},
			Labels:  map[string]string{"com.docker.compose.service": "db"},
			Command: "docker-entrypoint.sh postgres",
			Stats: &ContainerStats{
				CPU:     ContainerCPU{Usage: 12.5, System: 5.0, Cores: 4, Throttling: 2.1},
				Memory:  ContainerMemory{Usage: 268435456, Limit: 1073741824, Cache: 134217728, RSS: 134217728, Swap: 0, MaxUsage: 402653184},
				Network: ContainerNetwork{RxBytes: 524288000, RxPackets: 200000, TxBytes: 1048576000, TxPackets: 350000},
				BlockIO: ContainerBlockIO{ReadBytes: 1073741824, WriteBytes: 536870912, ReadOps: 50000, WriteOps: 25000},
				PIDs:    15,
			},
			Health:        ContainerHealth{Status: HealthHealthy, FailingStreak: 0},
			RestartPolicy: RestartPolicy{Name: "unless-stopped", MaximumRetryCount: 0},
			ExitCode:      0,
			Logs:          []string{"2026-03-20T00:00:00.000Z LOG:  database system is ready to accept connections", "2026-03-20T00:00:01.000Z LOG:  autovacuum launcher started"},
		},
		{
			ID:      "def45678gh9012345678901234abcdef",
			Name:    "redis-cache",
			Image:   "redis:7-alpine",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-168 * time.Hour),
			Started: now.Add(-72 * time.Hour),
			Ports: []Port{
				{PrivatePort: 6379, PublicPort: 6379, Type: "tcp", IP: "0.0.0.0"},
			},
			Networks: []Network{
				{Name: "backend", IP: "172.18.0.3", Gateway: "172.18.0.1", MAC: "02:42:ac:12:00:03"},
			},
			Labels:  map[string]string{"com.docker.compose.service": "cache"},
			Command: "redis-server --appendonly yes",
			Stats: &ContainerStats{
				CPU:     ContainerCPU{Usage: 1.2, System: 0.5, Cores: 4, Throttling: 0},
				Memory:  ContainerMemory{Usage: 31457280, Limit: 268435456, Cache: 5242880, RSS: 26214400, Swap: 0, MaxUsage: 41943040},
				Network: ContainerNetwork{RxBytes: 209715200, RxPackets: 100000, TxBytes: 419430400, TxPackets: 150000},
				BlockIO: ContainerBlockIO{ReadBytes: 10485760, WriteBytes: 104857600, ReadOps: 500, WriteOps: 5000},
				PIDs:    4,
			},
			Health:        ContainerHealth{Status: HealthHealthy, FailingStreak: 0},
			RestartPolicy: RestartPolicy{Name: "always", MaximumRetryCount: 0},
			ExitCode:      0,
			Logs:          []string{"2026-03-20T00:00:00.000Z # Server initialized", "2026-03-20T00:00:01.000Z * Ready to accept connections on port 6379"},
		},
		{
			ID:       "ghi78901jk2345678901234567abcdef",
			Name:     "app-worker",
			Image:    "myapp:2.1.0",
			Status:   StatusExited,
			State:    "exited",
			Created:  now.Add(-24 * time.Hour),
			Started:  now.Add(-12 * time.Hour),
			Finished: now.Add(-1 * time.Hour),
			Networks: []Network{
				{Name: "backend", IP: "172.18.0.4", Gateway: "172.18.0.1", MAC: "02:42:ac:12:00:04"},
			},
			Labels:        map[string]string{"com.docker.compose.service": "worker", "env": "production"},
			Command:       "./worker --queue=default",
			Health:        ContainerHealth{Status: HealthNone},
			RestartPolicy: RestartPolicy{Name: "on-failure", MaximumRetryCount: 3},
			ExitCode:      1,
			ExitReason:    "app_crashed",
			OOMKilled:     false,
			Logs:          []string{"2026-03-23T03:00:00.000Z [info] Worker starting...", "2026-03-23T03:00:05.000Z [warn] Queue backlog detected", "2026-03-23T12:00:00.000Z [error] Connection lost to broker", "2026-03-23T12:00:01.000Z [fatal] Panic: runtime error"},
		},
		{
			ID:      "jkl01234mn5678901234567890abcdef",
			Name:    "monitoring-grafana",
			Image:   "grafana/grafana:10.2",
			Status:  StatusRunning,
			State:   "running",
			Created: now.Add(-96 * time.Hour),
			Started: now.Add(-48 * time.Hour),
			Ports: []Port{
				{PrivatePort: 3000, PublicPort: 3000, Type: "tcp", IP: "0.0.0.0"},
			},
			Mounts: []Mount{
				{Source: "/var/lib/grafana", Destination: "/var/lib/grafana", Type: "volume", Mode: "rw"},
			},
			Networks: []Network{
				{Name: "monitoring", IP: "172.19.0.2", Gateway: "172.19.0.1", MAC: "02:42:ac:13:00:02"},
			},
			Labels:  map[string]string{"com.docker.compose.service": "grafana"},
			Command: "/run.sh",
			Stats: &ContainerStats{
				CPU:     ContainerCPU{Usage: 8.7, System: 3.2, Cores: 4, Throttling: 0},
				Memory:  ContainerMemory{Usage: 157286400, Limit: 536870912, Cache: 52428800, RSS: 104857600, Swap: 0, MaxUsage: 209715200},
				Network: ContainerNetwork{RxBytes: 314572800, RxPackets: 120000, TxBytes: 629145600, TxPackets: 200000},
				BlockIO: ContainerBlockIO{ReadBytes: 209715200, WriteBytes: 104857600, ReadOps: 10000, WriteOps: 5000},
				PIDs:    12,
			},
			Health:        ContainerHealth{Status: HealthHealthy, FailingStreak: 0},
			RestartPolicy: RestartPolicy{Name: "always", MaximumRetryCount: 0},
			ExitCode:      0,
			Logs:          []string{"2026-03-21T00:00:00.000Z [info] Starting Grafana", "2026-03-21T00:00:02.000Z [info] HTTP Server Listen on :3000"},
		},
	}
}
