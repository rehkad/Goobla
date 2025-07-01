package store

import (
	"os"
	"path/filepath"
)

func getStorePath() string {
	if s := os.Getenv("GOOBLA_CONFIG"); s != "" {
		return s
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	return filepath.Join(localAppData, "Goobla", "config.json")
}
