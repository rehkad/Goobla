//go:build darwin || windows

package cmd

import (
	"context"
	"errors"
	"time"

	"github.com/goobla/goobla/api"
)

func waitForServer(ctx context.Context, client *api.Client) error {
	// wait for the server to start
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return errors.New("timed out waiting for server to start")
		case <-ticker.C:
			if err := client.Heartbeat(ctx); err == nil {
				return nil // server has started
			}
		}
	}
}
