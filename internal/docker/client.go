package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Client wraps the Docker Engine API for listing containers and streaming logs.
type Client struct {
	cli *client.Client
}

// Container is a simplified view returned to the UI.
type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// NewClient creates a Docker client. Empty host uses the default local socket.
func NewClient(host string) (*Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("new docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close releases the underlying Docker client.
func (c *Client) Close() error {
	return c.cli.Close()
}

// Ping verifies Docker Engine connectivity.
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := c.cli.Ping(ctx)
	return err
}

// ListContainers returns all containers (running and stopped).
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	items, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	out := make([]Container, 0, len(items))
	for _, item := range items {
		name := ""
		if len(item.Names) > 0 {
			name = item.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		out = append(out, Container{
			ID:      item.ID,
			Name:    name,
			Image:   item.Image,
			State:   item.State,
			Status:  item.Status,
			Created: time.Unix(item.Created, 0).UTC(),
			Labels:  item.Labels,
		})
	}
	return out, nil
}

// StreamLogs tails container logs. The returned reader yields multiplexed Docker log frames
// when Tty is false; callers should demux or treat as raw bytes as needed.
func (c *Client) StreamLogs(ctx context.Context, id string, follow bool, tail string) (io.ReadCloser, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: true,
		Tail:       tail,
	}
	rc, err := c.cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return nil, fmt.Errorf("container logs %s: %w", id, err)
	}
	return rc, nil
}

// InspectTty reports whether the container was started with a TTY (affects log framing).
func (c *Client) InspectTty(ctx context.Context, id string) (bool, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", id, err)
	}
	if info.Config == nil {
		return false, nil
	}
	return info.Config.Tty, nil
}
