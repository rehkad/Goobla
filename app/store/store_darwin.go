package store

import (
	"os"
	"path/filepath"

	"github.com/goobla/goobla/envconfig"
)

func getStorePath() string {
	if s := os.Getenv("GOOBLA_CONFIG"); s != "" {
		return s
	}

	dir := envconfig.ConfigDir()
	return filepath.Join(dir, "config.json")
}
