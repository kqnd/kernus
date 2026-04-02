package metrics

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Snapshot struct {
	CPUPercent  float64
	MemoryUsed  int64
	MemoryTotal int64
	DiskUsed    int64
	DiskTotal   int64
	Processes   []ProcessInfo
	CollectedAt time.Time
}

type ProcessInfo struct {
	Address string
	Port    int
	Name    string
}

func (s *Snapshot) FormatLine() string {
	memGB := float64(s.MemoryUsed) / (1024 * 1024 * 1024)
	memTotalGB := float64(s.MemoryTotal) / (1024 * 1024 * 1024)
	diskGB := float64(s.DiskUsed) / (1024 * 1024 * 1024)
	diskTotalGB := float64(s.DiskTotal) / (1024 * 1024 * 1024)
	return fmt.Sprintf("✓ %s | CPU: %.1f%% | MEM: %.1f/%.1fGB | DISK: %.0f/%.0fGB",
		s.CollectedAt.Format("15:04:05"),
		s.CPUPercent,
		memGB, memTotalGB,
		diskGB, diskTotalGB,
	)
}

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (co *Collector) Collect() (*Snapshot, error) {
	return Collect()
}

func Collect() (*Snapshot, error) {
	snap := &Snapshot{
		CollectedAt: time.Now(),
	}

	cpuPct, err := collectCPU()
	if err != nil {
		return nil, fmt.Errorf("collecting CPU: %w", err)
	}
	snap.CPUPercent = cpuPct

	memUsed, memTotal, err := collectMemory()
	if err != nil {
		return nil, fmt.Errorf("collecting memory: %w", err)
	}
	snap.MemoryUsed = memUsed
	snap.MemoryTotal = memTotal

	diskUsed, diskTotal, err := collectDisk()
	if err != nil {
		return nil, fmt.Errorf("collecting disk: %w", err)
	}
	snap.DiskUsed = diskUsed
	snap.DiskTotal = diskTotal

	procs, err := collectProcesses()
	if err != nil {
		snap.Processes = nil
	} else {
		snap.Processes = procs
	}

	return snap, nil
}

func collectCPU() (float64, error) {
	if runtime.GOOS != "linux" {
		return collectCPUFallback()
	}

	idle1, total1, err := readCPUStat()
	if err != nil {
		return collectCPUFallback()
	}

	time.Sleep(100 * time.Millisecond)

	idle2, total2, err := readCPUStat()
	if err != nil {
		return collectCPUFallback()
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta == 0 {
		return 0, nil
	}

	return (1.0 - idleDelta/totalDelta) * 100, nil
}

func readCPUStat() (idle uint64, total uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("unexpected /proc/stat format")
		}
		var values []uint64
		for _, field := range fields[1:] {
			val, parseErr := strconv.ParseUint(field, 10, 64)
			if parseErr != nil {
				return 0, 0, fmt.Errorf("parsing /proc/stat field: %w", parseErr)
			}
			values = append(values, val)
		}
		for _, v := range values {
			total += v
		}
		if len(values) > 3 {
			idle = values[3]
		}
		return idle, total, nil
	}
	return 0, 0, fmt.Errorf("/proc/stat: cpu line not found")
}

func collectCPUFallback() (float64, error) {
	return float64(runtime.NumCPU()) * 5, nil
}

func collectMemory() (used int64, total int64, err error) {
	if runtime.GOOS != "linux" {
		return collectMemoryFallback()
	}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return collectMemoryFallback()
	}
	defer f.Close()

	var memTotal, memAvailable int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMemInfoValue(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMemInfoValue(line)
		}
	}

	if memTotal == 0 {
		return collectMemoryFallback()
	}

	return memTotal - memAvailable, memTotal, nil
}

func parseMemInfoValue(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return val * 1024
}

func collectMemoryFallback() (int64, int64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc), int64(m.Sys), nil
}
