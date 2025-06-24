//go:build windows

package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoUpgradeStartFailure(t *testing.T) {
	oldStageDir := UpdateStageDir
	oldUpgradeLog := UpgradeLogFile
	oldExec := execCommand
	oldExit := osExit
	defer func() {
		UpdateStageDir = oldStageDir
		UpgradeLogFile = oldUpgradeLog
		execCommand = oldExec
		osExit = oldExit
	}()

	tmp := t.TempDir()
	subDir := filepath.Join(tmp, "1")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	installer := filepath.Join(subDir, "BadInstaller.exe")
	require.NoError(t, os.WriteFile(installer, []byte("oops"), 0o644))
	UpdateStageDir = tmp
	UpgradeLogFile = filepath.Join(tmp, "upgrade.log")

	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("nonexistent-cmd")
	}
	osExit = func(int) {}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	done <- 0

	err := DoUpgrade(cancel, done)
	require.Error(t, err)

	cancel()
}
