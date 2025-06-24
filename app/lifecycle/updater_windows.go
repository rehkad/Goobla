package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// osExit allows tests to stub out os.Exit.
var osExit = os.Exit
var execCommand = exec.Command

func DoUpgrade(cancel context.CancelFunc, done chan int) error {
	files, err := filepath.Glob(filepath.Join(UpdateStageDir, "*", "*.exe")) // TODO generalize for multiplatform
	if err != nil {
		return fmt.Errorf("failed to lookup downloads: %s", err)
	}
	if len(files) == 0 {
		return errors.New("no update downloads found")
	} else if len(files) > 1 {
		// Shouldn't happen
		slog.Warn(fmt.Sprintf("multiple downloads found, using first one %v", files))
	}
	installerExe := files[0]

	slog.Info("starting upgrade with " + installerExe)
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

	slog.Debug(fmt.Sprintf("starting installer: %s %v", installerExe, installArgs))
	os.Chdir(filepath.Dir(UpgradeLogFile)) //nolint:errcheck
	cmd := execCommand(installerExe, installArgs...)

	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		if stderr.Len() > 0 {
			slog.Error("installer stderr", "output", stderr.String())
		}
		return fmt.Errorf("unable to start goobla app %w", err)
	}

	var waitErr error
	doneWait := make(chan error, 1)
	go func() { doneWait <- cmd.Wait() }()
	select {
	case waitErr = <-doneWait:
		exitCode := 0
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		if stderr.Len() > 0 {
			slog.Error("installer stderr", "output", stderr.String())
		}
		return fmt.Errorf("installer exited with code %d", exitCode)
	case <-time.After(200 * time.Millisecond):
		// assume running
	}

	if cmd.Process != nil {
		err = cmd.Process.Release()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to release server process: %s", err))
		}
	} else {
		if stderr.Len() > 0 {
			slog.Error("installer stderr", "output", stderr.String())
		}
		return errors.New("installer process did not start")
	}

	// lingering check done via Wait above

	slog.Info("Installer started in background, exiting")

	osExit(0)
	// Not reached
	return nil
}
