package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeKeypairCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := initializeKeypair(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	priv := filepath.Join(dir, ".goobla", "id_ed25519")
	pub := filepath.Join(dir, ".goobla", "id_ed25519.pub")

	if _, err := os.Stat(priv); err != nil {
		t.Fatalf("private key not created: %v", err)
	}
	if _, err := os.Stat(pub); err != nil {
		t.Fatalf("public key not created: %v", err)
	}

	info, err := os.Stat(filepath.Dir(priv))
	if err != nil {
		t.Fatalf("dir stat error: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("expected dir perm 0700, got %v", info.Mode().Perm())
	}

	// second call should not error
	if err := initializeKeypair(); err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
}
