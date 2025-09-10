package models

import (
	"fmt"
	"time"
)

type Machine struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      Status    `json:"status"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage Memory    `json:"memory_usage"`
	DiskUsage   Disk      `json:"disk_usage"`
	IP          string    `json:"ip"`
	LastSeen    time.Time `json:"last_seen"`
	Uptime      Duration  `json:"uptime"`
	Processes   []Process `json:"processes"`
	Group       string    `json:"group"`
}

type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusError   Status = "error"
)

func (s Status) Color() string {
	switch s {
	case StatusOnline:
		return "green"
	case StatusOffline:
		return "red"
	case StatusError:
		return "red"
	default:
		return "white"
	}
}

type Memory struct {
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

func (m Memory) Percentage() float64 {
	if m.Total == 0 {
		return 0
	}

	return float64(m.Used) / float64(m.Total) * 100
}

func (m Memory) String() string {
	return fmt.Sprintf("%.1fGB / %.1fGB", float64(m.Used)/1024/1024/1024, float64(m.Total)/1024/1024/1024)
}

type Disk struct {
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

func (d Disk) Percentage() float64 {
	if d.Total == 0 {
		return 0
	}
	return float64(d.Used) / float64(d.Total) * 100
}

func (d Disk) String() string {
	return fmt.Sprintf("%.0fGB / %.0fGB",
		float64(d.Used)/1024/1024/1024,
		float64(d.Total)/1024/1024/1024)
}

type Duration struct {
	Seconds int64 `json:"seconds"`
}

func (d Duration) String() string {
	days := d.Seconds / 86400
	hours := (d.Seconds % 86400) / 3600
	minutes := (d.Seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

type Process struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Name    string `json:"name"`
}

type Group struct {
	Name     string `json:"name"`
	Machines int    `json:"machine_count"`
}

func MockMachines() []Machine {
	return []Machine{
		{
			ID:       "m1",
			Name:     "web-server-01",
			Status:   StatusOnline,
			CPUUsage: 23.5,
			MemoryUsage: Memory{
				Used:  4 * 1024 * 1024 * 1024,
				Total: 8 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  120 * 1024 * 1024 * 1024,
				Total: 256 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.10",
			LastSeen: time.Now().Add(-2 * time.Minute),
			Uptime:   Duration{Seconds: 86400 * 2}, // 2 dias
			Processes: []Process{
				{Name: "nginx", Address: "127.0.0.1", Port: 80},
				{Name: "app", Address: "127.0.0.1", Port: 8080},
			},
			Group: "frontend",
		},
		{
			ID:       "m2",
			Name:     "web-server-02",
			Status:   StatusOnline,
			CPUUsage: 55.3,
			MemoryUsage: Memory{
				Used:  6 * 1024 * 1024 * 1024,
				Total: 8 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  200 * 1024 * 1024 * 1024,
				Total: 512 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.11",
			LastSeen: time.Now().Add(-10 * time.Second),
			Uptime:   Duration{Seconds: 86400*5 + 3600*3}, // 5 dias 3h
			Processes: []Process{
				{Name: "nginx", Address: "127.0.0.1", Port: 80},
				{Name: "worker", Address: "127.0.0.1", Port: 9000},
			},
			Group: "frontend",
		},
		{
			ID:       "m3",
			Name:     "db-server-01",
			Status:   StatusOnline,
			CPUUsage: 73.2,
			MemoryUsage: Memory{
				Used:  12 * 1024 * 1024 * 1024,
				Total: 16 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  800 * 1024 * 1024 * 1024,
				Total: 1024 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.20",
			LastSeen: time.Now().Add(-5 * time.Second),
			Uptime:   Duration{Seconds: 86400 * 15}, // 15 dias
			Processes: []Process{
				{Name: "postgres", Address: "127.0.0.1", Port: 5432},
			},
			Group: "database",
		},
		{
			ID:       "m4",
			Name:     "db-server-02",
			Status:   StatusOffline,
			CPUUsage: 90.1,
			MemoryUsage: Memory{
				Used:  15 * 1024 * 1024 * 1024,
				Total: 16 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  950 * 1024 * 1024 * 1024,
				Total: 1024 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.21",
			LastSeen: time.Now().Add(-30 * time.Second),
			Uptime:   Duration{Seconds: 86400 * 20},
			Processes: []Process{
				{Name: "mysql", Address: "127.0.0.1", Port: 3306},
			},
			Group: "database",
		},
		{
			ID:       "m5",
			Name:     "cache-01",
			Status:   StatusOnline,
			CPUUsage: 12.7,
			MemoryUsage: Memory{
				Used:  2 * 1024 * 1024 * 1024,
				Total: 4 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  20 * 1024 * 1024 * 1024,
				Total: 64 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.30",
			LastSeen: time.Now().Add(-1 * time.Minute),
			Uptime:   Duration{Seconds: 3600 * 10}, // 10h
			Processes: []Process{
				{Name: "redis", Address: "127.0.0.1", Port: 6379},
			},
			Group: "cache",
		},
		{
			ID:       "m6",
			Name:     "cache-02",
			Status:   StatusOffline,
			CPUUsage: 0,
			MemoryUsage: Memory{
				Used:  0,
				Total: 4 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  0,
				Total: 64 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.31",
			LastSeen: time.Now().Add(-2 * time.Hour),
			Uptime:   Duration{Seconds: 0},
			Processes: []Process{
				{Name: "redis", Address: "127.0.0.1", Port: 6379},
			},
			Group: "cache",
		},
		{
			ID:       "m7",
			Name:     "worker-01",
			Status:   StatusOnline,
			CPUUsage: 45.6,
			MemoryUsage: Memory{
				Used:  3 * 1024 * 1024 * 1024,
				Total: 8 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  100 * 1024 * 1024 * 1024,
				Total: 256 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.40",
			LastSeen: time.Now().Add(-15 * time.Second),
			Uptime:   Duration{Seconds: 3600 * 72}, // 3 dias
			Processes: []Process{
				{Name: "worker", Address: "127.0.0.1", Port: 9000},
			},
			Group: "backend",
		},
		{
			ID:       "m8",
			Name:     "worker-02",
			Status:   StatusOnline,
			CPUUsage: 33.2,
			MemoryUsage: Memory{
				Used:  5 * 1024 * 1024 * 1024,
				Total: 8 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  120 * 1024 * 1024 * 1024,
				Total: 256 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.41",
			LastSeen: time.Now().Add(-50 * time.Second),
			Uptime:   Duration{Seconds: 3600 * 150}, // ~6 dias
			Processes: []Process{
				{Name: "worker", Address: "127.0.0.1", Port: 9000},
			},
			Group: "backend",
		},
		{
			ID:       "m9",
			Name:     "monitoring-01",
			Status:   StatusOnline,
			CPUUsage: 18.9,
			MemoryUsage: Memory{
				Used:  2 * 1024 * 1024 * 1024,
				Total: 4 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  50 * 1024 * 1024 * 1024,
				Total: 200 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.50",
			LastSeen: time.Now(),
			Uptime:   Duration{Seconds: 3600 * 240}, // 10 dias
			Processes: []Process{
				{Name: "prometheus", Address: "127.0.0.1", Port: 9090},
				{Name: "grafana", Address: "127.0.0.1", Port: 3000},
			},
			Group: "monitoring",
		},
		{
			ID:       "m10",
			Name:     "backup-01",
			Status:   StatusOffline,
			CPUUsage: 0,
			MemoryUsage: Memory{
				Used:  0,
				Total: 8 * 1024 * 1024 * 1024,
			},
			DiskUsage: Disk{
				Used:  1024 * 1024 * 1024 * 1024,
				Total: 2 * 1024 * 1024 * 1024 * 1024,
			},
			IP:       "192.168.0.60",
			LastSeen: time.Now().Add(-24 * time.Hour),
			Uptime:   Duration{Seconds: 0},
			Processes: []Process{
				{Name: "backup-service", Address: "127.0.0.1", Port: 7000},
			},
			Group: "backup",
		},
	}
}
