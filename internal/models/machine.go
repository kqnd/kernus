package models

import (
	"fmt"
	"time"
)

type MachineStatus string

const (
	MachineOnline  MachineStatus = "online"
	MachineOffline MachineStatus = "offline"
	MachineError   MachineStatus = "error"
)

func (s MachineStatus) Color() string {
	switch s {
	case MachineOnline:
		return "green"
	case MachineOffline:
		return "red"
	case MachineError:
		return "yellow"
	default:
		return "white"
	}
}

type Machine struct {
	ID          string
	Name        string
	Status      MachineStatus
	CPUUsage    float64
	MemoryUsage Memory
	DiskUsage   Disk
	IP          string
	LastSeen    time.Time
	Uptime      Duration
	Processes   []Process
	Group       string
}

type Memory struct {
	Used  int64
	Total int64
}

func (m *Memory) Percentage() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(m.Used) / float64(m.Total) * 100
}

func (m *Memory) String() string {
	return fmt.Sprintf("%s / %s",
		FormatBytesHuman(m.Used),
		FormatBytesHuman(m.Total),
	)
}

type Disk struct {
	Used  int64
	Total int64
}

func (d *Disk) Percentage() float64 {
	if d.Total == 0 {
		return 0
	}
	return float64(d.Used) / float64(d.Total) * 100
}

func (d *Disk) String() string {
	return fmt.Sprintf("%s / %s",
		FormatBytesHuman(d.Used),
		FormatBytesHuman(d.Total),
	)
}

type Duration struct {
	Seconds int64
}

func (d *Duration) String() string {
	total := d.Seconds
	days := total / 86400
	hours := (total % 86400) / 3600
	if days > 0 {
		return fmt.Sprintf("%d days, %d hours", days, hours)
	}
	minutes := (total % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

type Process struct {
	Address string
	Port    int
	Name    string
}

type Group struct {
	Name         string
	MachineCount int
}

func MockMachines() []Machine {
	now := time.Now()
	return []Machine{
		{
			ID: "m-001", Name: "web-server-01", Status: MachineOnline, CPUUsage: 23.5,
			MemoryUsage: Memory{Used: 4294967296, Total: 8589934592},
			DiskUsage:   Disk{Used: 128849018880, Total: 274877906944},
			IP:          "192.168.0.10", LastSeen: now, Uptime: Duration{Seconds: 172800},
			Processes: []Process{{Address: "0.0.0.0", Port: 80, Name: "nginx"}, {Address: "0.0.0.0", Port: 443, Name: "nginx"}},
			Group:     "frontend",
		},
		{
			ID: "m-002", Name: "web-server-02", Status: MachineOnline, CPUUsage: 55.3,
			MemoryUsage: Memory{Used: 6442450944, Total: 8589934592},
			DiskUsage:   Disk{Used: 171798691840, Total: 274877906944},
			IP:          "192.168.0.11", LastSeen: now, Uptime: Duration{Seconds: 86400},
			Processes: []Process{{Address: "0.0.0.0", Port: 80, Name: "nginx"}},
			Group:     "frontend",
		},
		{
			ID: "m-003", Name: "db-server-01", Status: MachineOnline, CPUUsage: 45.2,
			MemoryUsage: Memory{Used: 12884901888, Total: 17179869184},
			DiskUsage:   Disk{Used: 429496729600, Total: 536870912000},
			IP:          "192.168.1.10", LastSeen: now, Uptime: Duration{Seconds: 604800},
			Processes: []Process{{Address: "0.0.0.0", Port: 5432, Name: "postgres"}, {Address: "127.0.0.1", Port: 9090, Name: "pg_exporter"}},
			Group:     "database",
		},
		{
			ID: "m-004", Name: "db-server-02", Status: MachineOffline, CPUUsage: 0,
			MemoryUsage: Memory{Used: 0, Total: 17179869184},
			DiskUsage:   Disk{Used: 214748364800, Total: 536870912000},
			IP:          "192.168.1.11", LastSeen: now.Add(-5 * time.Minute), Uptime: Duration{Seconds: 0},
			Group: "database",
		},
		{
			ID: "m-005", Name: "cache-01", Status: MachineOnline, CPUUsage: 12.8,
			MemoryUsage: Memory{Used: 3221225472, Total: 4294967296},
			DiskUsage:   Disk{Used: 21474836480, Total: 107374182400},
			IP:          "192.168.2.10", LastSeen: now, Uptime: Duration{Seconds: 259200},
			Processes: []Process{{Address: "0.0.0.0", Port: 6379, Name: "redis-server"}},
			Group:     "cache",
		},
		{
			ID: "m-006", Name: "cache-02", Status: MachineOffline, CPUUsage: 0,
			MemoryUsage: Memory{Used: 0, Total: 4294967296},
			DiskUsage:   Disk{Used: 10737418240, Total: 107374182400},
			IP:          "192.168.2.11", LastSeen: now.Add(-10 * time.Minute), Uptime: Duration{Seconds: 0},
			Group: "cache",
		},
		{
			ID: "m-007", Name: "api-server-01", Status: MachineOnline, CPUUsage: 67.8,
			MemoryUsage: Memory{Used: 7516192768, Total: 8589934592},
			DiskUsage:   Disk{Used: 85899345920, Total: 274877906944},
			IP:          "192.168.3.10", LastSeen: now, Uptime: Duration{Seconds: 432000},
			Processes: []Process{{Address: "0.0.0.0", Port: 8080, Name: "java"}, {Address: "0.0.0.0", Port: 9100, Name: "node_exporter"}},
			Group:     "backend",
		},
		{
			ID: "m-008", Name: "api-server-02", Status: MachineError, CPUUsage: 95.2,
			MemoryUsage: Memory{Used: 8374067200, Total: 8589934592},
			DiskUsage:   Disk{Used: 257698037760, Total: 274877906944},
			IP:          "192.168.3.11", LastSeen: now.Add(-30 * time.Second), Uptime: Duration{Seconds: 432000},
			Processes: []Process{{Address: "0.0.0.0", Port: 8080, Name: "java"}},
			Group:     "backend",
		},
		{
			ID: "m-009", Name: "prometheus", Status: MachineOnline, CPUUsage: 18.3,
			MemoryUsage: Memory{Used: 2147483648, Total: 4294967296},
			DiskUsage:   Disk{Used: 107374182400, Total: 214748364800},
			IP:          "192.168.4.10", LastSeen: now, Uptime: Duration{Seconds: 604800},
			Processes: []Process{{Address: "0.0.0.0", Port: 9090, Name: "prometheus"}},
			Group:     "monitoring",
		},
		{
			ID: "m-010", Name: "grafana", Status: MachineOnline, CPUUsage: 8.5,
			MemoryUsage: Memory{Used: 1073741824, Total: 2147483648},
			DiskUsage:   Disk{Used: 32212254720, Total: 107374182400},
			IP:          "192.168.4.11", LastSeen: now, Uptime: Duration{Seconds: 604800},
			Processes: []Process{{Address: "0.0.0.0", Port: 3000, Name: "grafana-server"}},
			Group:     "monitoring",
		},
	}
}
