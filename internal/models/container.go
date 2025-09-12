package models

import (
	"fmt"
	"strings"
	"time"
)

type Container struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Status   ContainerStatus   `json:"status"`
	State    string            `json:"state"`
	Created  time.Time         `json:"created"`
	Started  time.Time         `json:"started"`
	Ports    []Port            `json:"ports"`
	Mounts   []Mount           `json:"mounts"`
	Networks []Network         `json:"networks"`
	Labels   map[string]string `json:"labels"`
	Command  string            `json:"command"`
	CPUUsage float64           `json:"cpu_usage"`
	Memory   ContainerMemory   `json:"memory"`
}

type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusExited  ContainerStatus = "exited"
	StatusPaused  ContainerStatus = "paused"
	StatusStopped ContainerStatus = "stopped"
	StatusCreated ContainerStatus = "created"
	StatusDead    ContainerStatus = "dead"
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
}

func (m Mount) String() string {
	return fmt.Sprintf("%s:%s", m.Source, m.Destination)
}

type Network struct {
	Name      string `json:"name"`
	NetworkID string `json:"network_id"`
	IPAddress string `json:"ip_address"`
}

type ContainerMemory struct {
	Usage int64 `json:"usage"`
	Limit int64 `json:"limit"`
}

func (m ContainerMemory) Percentage() float64 {
	if m.Limit == 0 {
		return 0
	}
	return float64(m.Usage) / float64(m.Limit) * 100
}

func (m ContainerMemory) String() string {
	return fmt.Sprintf("%.1fMB / %.1fMB",
		float64(m.Usage)/1024/1024,
		float64(m.Limit)/1024/1024)
}

func (c *Container) ShortID() string {
	if len(c.ID) > 12 {
		return c.ID[:12]
	}
	return c.ID
}

func (c *Container) ShortName() string {
	name := c.Name
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
	return name
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

func (c *Container) ShortPort() string {
	if len(c.Ports) == 0 {
		return "No ports"
	}

	for _, port := range c.Ports {
		if port.PublicPort > 0 {
			return fmt.Sprintf("%d", port.PublicPort)
		}
	}

	return fmt.Sprintf("%d", c.Ports[0].PublicPort)
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
			},
			Command:  "nginx -g 'daemon off;'",
			CPUUsage: 5.2,
			Memory: ContainerMemory{
				Usage: 50 * 1024 * 1024,
				Limit: 512 * 1024 * 1024,
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
			Command:  "docker-entrypoint.sh postgres",
			CPUUsage: 15.7,
			Memory: ContainerMemory{
				Usage: 200 * 1024 * 1024,
				Limit: 1024 * 1024 * 1024,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "database",
			},
		},
		{
			ID:      "ghi789012345",
			Name:    "/redis-cache",
			Image:   "redis:6-alpine",
			Status:  StatusExited,
			State:   "exited",
			Created: now.Add(-6 * time.Hour),
			Started: time.Time{},
			Ports: []Port{
				{PrivatePort: 6379, Type: "tcp"},
			},
			Command:  "redis-server",
			CPUUsage: 0,
			Memory: ContainerMemory{
				Usage: 0,
				Limit: 256 * 1024 * 1024,
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
			Command:  "npm start",
			CPUUsage: 25.4,
			Memory: ContainerMemory{
				Usage: 150 * 1024 * 1024,
				Limit: 512 * 1024 * 1024,
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
			Command:  "/run.sh",
			CPUUsage: 0,
			Memory: ContainerMemory{
				Usage: 80 * 1024 * 1024,
				Limit: 256 * 1024 * 1024,
			},
			Labels: map[string]string{
				"com.docker.compose.service": "monitoring",
			},
		},
	}
}
