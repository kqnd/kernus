package agent

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type MockCollector struct {
	mu         sync.Mutex
	containers []mockContainer
	cycle      int
}

type mockContainer struct {
	id           string
	name         string
	behavior     mockBehavior
	status       string
	health       string
	cpuPercent   float32
	memoryUsed   uint64
	memoryLimit  uint64
	restartCount uint16
	memLeakBase  uint64
	spikePhase   int
	flapCounter  int
	crashCounter int
}

type mockBehavior int

const (
	behaviorCPUSpike      mockBehavior = iota
	behaviorMemoryLeak
	behaviorHealthFlap
	behaviorRestartStorm
	behaviorStable
	behaviorIdle
	behaviorCPUBurn
	behaviorHighMemory
	behaviorStatusChaos
	behaviorGradualRampUp
)

func NewMockCollector() *MockCollector {
	mc := &MockCollector{
		containers: generateMockContainers(),
	}
	return mc
}

func (mc *MockCollector) Close() error {
	return nil
}

func (mc *MockCollector) Collect(ctx context.Context) ([]ContainerMetric, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cycle++

	metrics := make([]ContainerMetric, 0, len(mc.containers))
	for i := range mc.containers {
		mc.updateContainer(&mc.containers[i])
		m := ContainerMetric{
			ID:           mc.containers[i].id,
			Name:         mc.containers[i].name,
			CPUPercent:   mc.containers[i].cpuPercent,
			MemoryUsed:   mc.containers[i].memoryUsed,
			MemoryLimit:  mc.containers[i].memoryLimit,
			RestartCount: mc.containers[i].restartCount,
			Status:       mc.containers[i].status,
			Health:       mc.containers[i].health,
			Timestamp:    time.Now().UTC(),
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (mc *MockCollector) updateContainer(c *mockContainer) {
	switch c.behavior {
	case behaviorCPUSpike:
		mc.updateCPUSpike(c)
	case behaviorMemoryLeak:
		mc.updateMemoryLeak(c)
	case behaviorHealthFlap:
		mc.updateHealthFlap(c)
	case behaviorRestartStorm:
		mc.updateRestartStorm(c)
	case behaviorStable:
		mc.updateStable(c)
	case behaviorIdle:
		mc.updateIdle(c)
	case behaviorCPUBurn:
		mc.updateCPUBurn(c)
	case behaviorHighMemory:
		mc.updateHighMemory(c)
	case behaviorStatusChaos:
		mc.updateStatusChaos(c)
	case behaviorGradualRampUp:
		mc.updateGradualRampUp(c)
	}
}

func (mc *MockCollector) updateCPUSpike(c *mockContainer) {
	c.spikePhase++
	c.status = "running"
	c.health = "healthy"
	if c.spikePhase%6 < 3 {
		c.cpuPercent = float32(85 + rand.Intn(16))
	} else {
		c.cpuPercent = float32(rand.Intn(5))
	}
	c.memoryUsed = 128*1024*1024 + uint64(rand.Int63n(64*1024*1024))
}

func (mc *MockCollector) updateMemoryLeak(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.cpuPercent = float32(10 + rand.Intn(15))

	growth := uint64(20+rand.Intn(60)) * 1024 * 1024
	c.memoryUsed += growth

	if c.memoryUsed > c.memoryLimit*95/100 {
		c.memoryUsed = c.memLeakBase
		c.restartCount++
		c.health = "unhealthy"
		if c.restartCount > 100 {
			c.restartCount = 100
		}
	}
}

func (mc *MockCollector) updateHealthFlap(c *mockContainer) {
	c.status = "running"
	c.cpuPercent = float32(20 + rand.Intn(30))
	c.memoryUsed = 256*1024*1024 + uint64(rand.Int63n(128*1024*1024))

	c.flapCounter++
	switch {
	case c.flapCounter%8 < 2:
		c.health = "unhealthy"
	case c.flapCounter%8 < 4:
		c.health = "starting"
	default:
		c.health = "healthy"
	}
}

func (mc *MockCollector) updateRestartStorm(c *mockContainer) {
	c.crashCounter++
	c.cpuPercent = float32(rand.Intn(100))
	c.memoryUsed = uint64(rand.Int63n(int64(c.memoryLimit)))

	switch c.crashCounter % 5 {
	case 0:
		c.status = "running"
		c.health = "starting"
	case 1:
		c.status = "running"
		c.health = "unhealthy"
	case 2:
		c.status = "exited"
		c.health = "none"
		c.restartCount++
	case 3:
		c.status = "restarting"
		c.health = "none"
	case 4:
		c.status = "created"
		c.health = "none"
	}
	if c.restartCount > 500 {
		c.restartCount = 500
	}
}

func (mc *MockCollector) updateStable(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.cpuPercent = float32(15+rand.Intn(10)) + float32(rand.Intn(10))/10.0
	base := uint64(200 * 1024 * 1024)
	c.memoryUsed = base + uint64(rand.Int63n(20*1024*1024))
}

func (mc *MockCollector) updateIdle(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.cpuPercent = float32(rand.Intn(2)) + float32(rand.Intn(10))/10.0
	c.memoryUsed = 10*1024*1024 + uint64(rand.Int63n(5*1024*1024))
}

func (mc *MockCollector) updateCPUBurn(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.cpuPercent = float32(88 + rand.Intn(13))
	c.memoryUsed = 512*1024*1024 + uint64(rand.Int63n(256*1024*1024))
}

func (mc *MockCollector) updateHighMemory(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.cpuPercent = float32(5 + rand.Intn(20))
	minMem := c.memoryLimit * 80 / 100
	rangeMem := c.memoryLimit * 15 / 100
	c.memoryUsed = minMem + uint64(rand.Int63n(int64(rangeMem)))
}

func (mc *MockCollector) updateStatusChaos(c *mockContainer) {
	statuses := []string{"running", "exited", "paused", "restarting", "created", "dead"}
	c.status = statuses[rand.Intn(len(statuses))]

	if c.status == "running" {
		healths := []string{"healthy", "unhealthy", "starting"}
		c.health = healths[rand.Intn(len(healths))]
		c.cpuPercent = float32(rand.Intn(100))
		c.memoryUsed = uint64(rand.Int63n(int64(c.memoryLimit)))
	} else {
		c.health = "none"
		c.cpuPercent = 0
		c.memoryUsed = 0
	}
}

func (mc *MockCollector) updateGradualRampUp(c *mockContainer) {
	c.status = "running"
	c.health = "healthy"
	c.spikePhase++

	progress := float32(c.spikePhase%60) / 60.0
	c.cpuPercent = progress*95 + float32(rand.Intn(5))
	if c.cpuPercent > 100 {
		c.cpuPercent = 100
	}
	c.memoryUsed = uint64(float64(c.memoryLimit) * float64(progress) * 0.9)

	if progress > 0.95 {
		c.health = "unhealthy"
	}
}

func generateMockContainers() []mockContainer {
	return []mockContainer{
		{
			id: "mock-cpu-spike-aa11bb22cc33", name: "api-gateway",
			behavior: behaviorCPUSpike, status: "running", health: "healthy",
			memoryLimit: 1024 * 1024 * 1024,
		},
		{
			id: "mock-mem-leak-dd44ee55ff66", name: "data-processor",
			behavior: behaviorMemoryLeak, status: "running", health: "healthy",
			memoryLimit: 2 * 1024 * 1024 * 1024, memLeakBase: 64 * 1024 * 1024,
			memoryUsed: 64 * 1024 * 1024,
		},
		{
			id: "mock-health-flap-1122334455", name: "payment-service",
			behavior: behaviorHealthFlap, status: "running", health: "healthy",
			memoryLimit: 512 * 1024 * 1024,
		},
		{
			id: "mock-restart-storm-6677889900", name: "worker-queue",
			behavior: behaviorRestartStorm, status: "running", health: "starting",
			memoryLimit: 768 * 1024 * 1024,
		},
		{
			id: "mock-stable-aabbccddee01", name: "nginx-proxy",
			behavior: behaviorStable, status: "running", health: "healthy",
			memoryLimit: 512 * 1024 * 1024,
		},
		{
			id: "mock-idle-aabbccddee02", name: "cron-scheduler",
			behavior: behaviorIdle, status: "running", health: "healthy",
			memoryLimit: 256 * 1024 * 1024,
		},
		{
			id: "mock-cpu-burn-aabbccddee03", name: "ml-training",
			behavior: behaviorCPUBurn, status: "running", health: "healthy",
			memoryLimit: 4 * 1024 * 1024 * 1024,
		},
		{
			id: "mock-high-mem-aabbccddee04", name: "postgres-primary",
			behavior: behaviorHighMemory, status: "running", health: "healthy",
			memoryLimit: 8 * 1024 * 1024 * 1024,
		},
		{
			id: "mock-chaos-aabbccddee05", name: "legacy-service",
			behavior: behaviorStatusChaos, status: "running", health: "healthy",
			memoryLimit: 1024 * 1024 * 1024,
		},
		{
			id: "mock-ramp-aabbccddee06", name: "batch-indexer",
			behavior: behaviorGradualRampUp, status: "running", health: "healthy",
			memoryLimit: 2 * 1024 * 1024 * 1024,
		},
		{
			id: "mock-stable-redis-aabb07", name: "redis-cache",
			behavior: behaviorStable, status: "running", health: "healthy",
			memoryLimit: 1024 * 1024 * 1024,
		},
		{
			id: "mock-idle-sidecar-aabb08", name: "log-collector",
			behavior: behaviorIdle, status: "running", health: "healthy",
			memoryLimit: 128 * 1024 * 1024,
		},
	}
}
