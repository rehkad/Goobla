package store

import (
	"os"
	"path/filepath"
)

func getStorePath() string {
	if s := os.Getenv("GOOBLA_CONFIG"); s != "" {
		return s
	}

	if os.Geteuid() == 0 {
		return "/etc/goobla/config.json"
	}

	home := os.Getenv("HOME")
	return filepath.Join(home, ".goobla", "config.json")
}
