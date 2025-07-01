package store

import (
	"os"
	"path/filepath"
)

func getStorePath() string {
	if s := os.Getenv("GOOBLA_CONFIG"); s != "" {
		return s
	}

	home := os.Getenv("HOME")
	return filepath.Join(home, "Library", "Application Support", "Goobla", "config.json")
}
