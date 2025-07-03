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

	if os.Geteuid() == 0 {
		return "/etc/goobla/config.json"
	}

	dir := envconfig.ConfigDir()
	return filepath.Join(dir, "config.json")
}
