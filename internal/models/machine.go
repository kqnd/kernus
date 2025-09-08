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
