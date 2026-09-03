package dockerctl

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Controller struct {
	client    *http.Client
	allowlist map[string]bool
}

func New(allowedContainers []string) *Controller {
	allowlist := make(map[string]bool, len(allowedContainers))
	for _, name := range allowedContainers {
		allowlist[name] = true
	}
	return &Controller{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", "/var/run/docker.sock")
				},
			},
		},
		allowlist: allowlist,
	}
}

func (c *Controller) action(ctx context.Context, container, verb string) error {
	if !c.allowlist[container] {
		return fmt.Errorf("container %q is not in the allowlist", container)
	}
	url := fmt.Sprintf("http://unix/containers/%s/%s", container, verb)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("docker api returned %d for %s %s", resp.StatusCode, verb, container)
	}
	return nil
}

func (c *Controller) Restart(ctx context.Context, container string) error {
	return c.action(ctx, container, "restart")
}
func (c *Controller) Stop(ctx context.Context, container string) error {
	return c.action(ctx, container, "stop")
}
func (c *Controller) Start(ctx context.Context, container string) error {
	return c.action(ctx, container, "start")
}

func (c *Controller) Allowed() []string {
	names := make([]string, 0, len(c.allowlist))
	for name := range c.allowlist {
		names = append(names, name)
	}
	return names
}
