package lifecycle

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	AppName          = "ollama app"
	CLIName          = "moogla"
	AppDir           = "/opt/Moogla"
	AppDataDir       = "/opt/Moogla"
	LogDir           = "/tmp"
	UpdateStageDir   = "/tmp"
	AppLogFile       = filepath.Join(LogDir, "app.log")
	ServerLogFile    = filepath.Join(LogDir, "server.log")
	UpgradeLogFile   = filepath.Join(LogDir, "upgrade.log")
	Installer        = "MooglaSetup.exe"
	LogRotationCount = 5
)

func init() {
	if runtime.GOOS == "windows" {
		AppName += ".exe"
		CLIName += ".exe"
		// Logs, configs, downloads go to LOCALAPPDATA
		localAppData := os.Getenv("LOCALAPPDATA")
		AppDataDir = filepath.Join(localAppData, "Moogla")
		LogDir = AppDataDir
		UpdateStageDir = filepath.Join(AppDataDir, "updates")
		AppLogFile = filepath.Join(LogDir, "app.log")
		ServerLogFile = filepath.Join(LogDir, "server.log")
		UpgradeLogFile = filepath.Join(LogDir, "upgrade.log")

		exe, err := os.Executable()
		if err != nil {
			slog.Warn("error discovering executable directory", "error", err)
			AppDir = filepath.Join(localAppData, "Programs", "Moogla")
		} else {
			AppDir = filepath.Dir(exe)
		}

		// Make sure we have PATH set correctly for any spawned children
		paths := strings.Split(os.Getenv("PATH"), ";")
		// Start with whatever we find in the PATH/LD_LIBRARY_PATH
		found := false
		for _, path := range paths {
			d, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			if strings.EqualFold(AppDir, d) {
				found = true
			}
		}
		if !found {
			paths = append(paths, AppDir)

			pathVal := strings.Join(paths, ";")
			slog.Debug("setting PATH=" + pathVal)
			err := os.Setenv("PATH", pathVal)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to update PATH: %s", err))
			}
		}

		// Make sure our logging dir exists
		_, err = os.Stat(AppDataDir)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(AppDataDir, 0o755); err != nil {
				slog.Error(fmt.Sprintf("create ollama dir %s: %v", AppDataDir, err))
			}
		}
	} else if runtime.GOOS == "darwin" {
		AppName += ".app"
		home, err := os.UserHomeDir()
		if err == nil {
			AppDataDir = filepath.Join(home, ".ollama")
			LogDir = filepath.Join(AppDataDir, "logs")
			UpdateStageDir = filepath.Join(AppDataDir, "updates")
			AppLogFile = filepath.Join(LogDir, "app.log")
			ServerLogFile = filepath.Join(LogDir, "server.log")
			UpgradeLogFile = filepath.Join(LogDir, "upgrade.log")
		}

		exe, err := os.Executable()
		if err != nil {
			slog.Warn("error discovering executable directory", "error", err)
		} else {
			AppDir = filepath.Dir(exe)
		}

		_, err = os.Stat(LogDir)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(LogDir, 0o755); err != nil {
				slog.Error(fmt.Sprintf("create ollama dir %s: %v", LogDir, err))
			}
		}
	} else if runtime.GOOS == "linux" {
		home, err := os.UserHomeDir()
		if err == nil {
			AppDataDir = filepath.Join(home, ".ollama")
			LogDir = filepath.Join(AppDataDir, "logs")
			UpdateStageDir = filepath.Join(AppDataDir, "updates")
			AppLogFile = filepath.Join(LogDir, "app.log")
			ServerLogFile = filepath.Join(LogDir, "server.log")
			UpgradeLogFile = filepath.Join(LogDir, "upgrade.log")
		}

		exe, err := os.Executable()
		if err != nil {
			slog.Warn("error discovering executable directory", "error", err)
		} else {
			AppDir = filepath.Dir(exe)
		}

		_, err = os.Stat(LogDir)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(LogDir, 0o755); err != nil {
				slog.Error(fmt.Sprintf("create ollama dir %s: %v", LogDir, err))
			}
		}
	}
}
