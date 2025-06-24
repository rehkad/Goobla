//go:build !windows && !darwin

package cmd

import (
	"context"
	"errors"

	"github.com/goobla/goobla/api"
)

func startApp(ctx context.Context, client *api.Client) error {
	return errors.New("could not connect to goobla server, run 'goobla serve' to start it")
}
