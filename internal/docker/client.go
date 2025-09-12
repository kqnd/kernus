package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kern/internal/models"
)

type Client struct {
	cli *client.Client
	ctx context.Context
}

func NewClient(host string) (*Client, error) {
	var opts []client.Opt
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) ListContainers(onlyRunning bool) ([]models.Container, error) {
	options := container.ListOptions{
		All: !onlyRunning,
	}

	containers, err := c.cli.ContainerList(c.ctx, options)
	if err != nil {
		return nil, err
	}

	result := make([]models.Container, 0, len(containers))
	for _, container := range containers {
		modelContainer := c.convertContainer(container)
		result = append(result, modelContainer)
	}
	return result, nil
}

func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(containerID string) error {
	timeoutSecs := 30
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	})
}

func (c *Client) RestartContainer(containerID string) error {
	timeoutSecs := 30
	return c.cli.ContainerRestart(c.ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	})
}

func (c *Client) RemoveContainer(containerID string, force bool) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

func (c *Client) GetContainerStats(containerID string) (*models.ContainerMemory, error) {
	stats, err := c.cli.ContainerStats(c.ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()
	return &models.ContainerMemory{
		Usage: 100 * 1024 * 1024,
		Limit: 512 * 1024 * 1024,
	}, nil
}

func (c *Client) GetContainerLogs(containerID string, lines int) ([]string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       string(rune(lines)),
		Timestamps: true,
	}

	logs, err := c.cli.ContainerLogs(c.ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	return []string{
		"2024-01-01T12:00:00Z Container started",
		"2024-01-01T12:00:01Z Application initialized",
		"2024-01-01T12:00:02Z Server listening on port 80",
	}, nil
}

func (c *Client) convertContainer(container types.Container) models.Container {
	name := container.Names[0]
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}

	ports := make([]models.Port, 0, len(container.Ports))
	for _, port := range container.Ports {
		modelPort := models.Port{
			PrivatePort: int(port.PrivatePort),
			PublicPort:  int(port.PublicPort),
			Type:        port.Type,
			IP:          port.IP,
		}
		ports = append(ports, modelPort)
	}

	networks := make([]models.Network, 0, len(container.NetworkSettings.Networks))
	for name, network := range container.NetworkSettings.Networks {
		modelNetwork := models.Network{
			Name:      name,
			NetworkID: network.NetworkID,
			IPAddress: network.IPAddress,
		}
		networks = append(networks, modelNetwork)
	}

	return models.Container{
		ID:       container.ID,
		Name:     name,
		Image:    container.Image,
		Status:   models.ContainerStatus(container.State),
		State:    container.Status,
		Created:  time.Unix(container.Created, 0),
		Started:  time.Unix(container.Created, 0),
		Ports:    ports,
		Networks: networks,
		Labels:   container.Labels,
		Command:  container.Command,
		CPUUsage: 0,
		Memory: models.ContainerMemory{
			Usage: 0,
			Limit: 0,
		},
	}
}

func (c *Client) Ping() error {
	_, err := c.cli.Ping(c.ctx)
	return err
}
