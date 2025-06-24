package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func DoUpgrade(cancel context.CancelFunc, done chan int) error {
	patterns := []string{"*.exe", "*.app", "*.sh"}
	var files []string
	for _, p := range patterns {
		m, err := filepath.Glob(filepath.Join(UpdateStageDir, "*", p))
		if err != nil {
			return fmt.Errorf("failed to lookup downloads: %w", err)
		}
		files = append(files, m...)
	}
	if len(files) == 0 {
		return errors.New("no update downloads found")
	} else if len(files) > 1 {
		// Shouldn't happen
		slog.Warn(fmt.Sprintf("multiple downloads found, using first one %v", files))
	}
	installerFile := files[0]

	slog.Info("starting upgrade with " + installerFile)
	slog.Info("upgrade log file " + UpgradeLogFile)

	// make the upgrade show progress, but non interactive
	installArgs := []string{
		"/CLOSEAPPLICATIONS",                    // Quit the tray app if it's still running
		"/LOG=" + filepath.Base(UpgradeLogFile), // Only relative seems reliable, so set pwd
		"/FORCECLOSEAPPLICATIONS",               // Force close the tray app - might be needed
		"/SP",                                   // Skip the "This will install... Do you wish to continue" prompt
		"/NOCANCEL",                             // Disable the ability to cancel upgrade mid-flight to avoid partially installed upgrades
		"/SILENT",
	}

	// Safeguard in case we have requests in flight that need to drain...
	slog.Info("Waiting for server to shutdown")
	cancel()
	if done != nil {
		<-done
	} else {
		// Shouldn't happen
		slog.Warn("done chan was nil, not actually waiting")
	}

	slog.Debug(fmt.Sprintf("starting installer: %s %v", installerFile, installArgs))
	os.Chdir(filepath.Dir(UpgradeLogFile)) //nolint:errcheck
	var cmd *exec.Cmd
	switch strings.ToLower(filepath.Ext(installerFile)) {
	case ".exe":
		cmd = exec.Command(installerFile, installArgs...)
	case ".app":
		cmd = exec.Command("open", installerFile)
	case ".sh":
		cmd = exec.Command("sh", installerFile)
	default:
		return fmt.Errorf("unsupported installer type %s", installerFile)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start goobla app %w", err)
	}

	if cmd.Process != nil {
		err = cmd.Process.Release()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to release server process: %s", err))
		}
	} else {
		// TODO - some details about why it didn't start, or is this a pedantic error case?
		return errors.New("installer process did not start")
	}

	// TODO should we linger for a moment and check to make sure it's actually running by checking the pid?

	slog.Info("Installer started in background, exiting")

	os.Exit(0)
	// Not reached
	return nil
}
