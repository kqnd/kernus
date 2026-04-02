package docker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/kiev/kernus/internal/models"
)

type MockClient struct {
	containers []models.Container
	mu         sync.RWMutex
	cycle      int
}

func NewMockClient() DockerClient {
	return &MockClient{
		containers: models.MockContainers(),
	}
}

func (m *MockClient) ListContainers(ctx context.Context, onlyRunning bool) ([]models.Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cycle++
	m.simulateDynamicBehaviors()

	if onlyRunning {
		var running []models.Container
		for _, c := range m.containers {
			if c.Status == models.StatusRunning {
				running = append(running, c)
			}
		}
		return running, nil
	}
	result := make([]models.Container, len(m.containers))
	copy(result, m.containers)
	return result, nil
}

func (m *MockClient) StartContainer(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			m.containers[i].Status = models.StatusRunning
			m.containers[i].State = "running"
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) StopContainer(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			m.containers[i].Status = models.StatusExited
			m.containers[i].State = "exited"
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) RestartContainer(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			m.containers[i].Status = models.StatusRunning
			m.containers[i].State = "running"
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) PauseContainer(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			m.containers[i].Status = models.StatusPaused
			m.containers[i].State = "paused"
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) UnpauseContainer(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			m.containers[i].Status = models.StatusRunning
			m.containers[i].State = "running"
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			if !force && c.Status == models.StatusRunning {
				return fmt.Errorf("container %s is running, use force to remove", containerID)
			}
			m.containers = append(m.containers[:i], m.containers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) GetContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			if c.Stats != nil {
				return c.Stats, nil
			}
			return nil, fmt.Errorf("no stats available for %s", containerID)
		}
	}
	return nil, fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.containers {
		if c.ID == containerID || c.ShortID() == containerID {
			if len(c.Logs) <= lines {
				return c.Logs, nil
			}
			return c.Logs[len(c.Logs)-lines:], nil
		}
	}
	return nil, fmt.Errorf("container %s not found", containerID)
}

func (m *MockClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockClient) Close() error {
	return nil
}

func (m *MockClient) simulateDynamicBehaviors() {
	for i := range m.containers {
		c := &m.containers[i]
		if c.Stats == nil || c.Status != models.StatusRunning {
			continue
		}

		switch {
		case containsName(c.Name, "web"):
			if m.cycle%4 < 2 {
				c.Stats.CPU.Usage = float64(80 + rand.Intn(21))
			} else {
				c.Stats.CPU.Usage = float64(2 + rand.Intn(8))
			}
			c.Stats.Memory.Usage = c.Stats.Memory.Usage + int64(rand.Intn(5*1024*1024)) - int64(rand.Intn(5*1024*1024))
			if c.Stats.Memory.Usage < 10*1024*1024 {
				c.Stats.Memory.Usage = 10 * 1024 * 1024
			}

		case containsName(c.Name, "postgres") || containsName(c.Name, "db"):
			c.Stats.CPU.Usage = float64(8 + rand.Intn(25))
			c.Stats.Memory.Usage += int64(rand.Intn(20 * 1024 * 1024))
			if c.Stats.Memory.Usage > c.Stats.Memory.Limit*90/100 {
				c.Stats.Memory.Usage = c.Stats.Memory.Limit * 60 / 100
			}
			c.Stats.CPU.Throttling = float64(rand.Intn(15))

		case containsName(c.Name, "redis") || containsName(c.Name, "cache"):
			c.Stats.CPU.Usage = float64(rand.Intn(5)) + float64(rand.Intn(10))/10.0
			if m.cycle%10 == 0 {
				c.Stats.CPU.Usage = float64(60 + rand.Intn(30))
			}

		case containsName(c.Name, "worker"):
			c.Stats.CPU.Usage = float64(30 + rand.Intn(50))
			if m.cycle%6 < 2 {
				c.Health.Status = models.HealthUnhealthy
				c.Health.FailingStreak++
			} else {
				c.Health.Status = models.HealthHealthy
				c.Health.FailingStreak = 0
			}

		case containsName(c.Name, "grafana") || containsName(c.Name, "monitoring"):
			c.Stats.CPU.Usage = float64(5+rand.Intn(10)) + float64(rand.Intn(10))/10.0
			c.Stats.Memory.Usage = int64(150+rand.Intn(30)) * 1024 * 1024

		default:
			c.Stats.CPU.Usage = float64(rand.Intn(100))
			c.Stats.Memory.Usage = int64(rand.Intn(int(c.Stats.Memory.Limit/1024/1024))) * 1024 * 1024
		}

		c.Stats.Network.RxBytes += int64(rand.Intn(10 * 1024 * 1024))
		c.Stats.Network.TxBytes += int64(rand.Intn(10 * 1024 * 1024))
		c.Stats.Network.RxPackets += int64(rand.Intn(5000))
		c.Stats.Network.TxPackets += int64(rand.Intn(5000))

		c.Stats.BlockIO.ReadBytes += int64(rand.Intn(5 * 1024 * 1024))
		c.Stats.BlockIO.WriteBytes += int64(rand.Intn(5 * 1024 * 1024))
	}
}

func containsName(name, substr string) bool {
	for i := 0; i <= len(name)-len(substr); i++ {
		if name[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
