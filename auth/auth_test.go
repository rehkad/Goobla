package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func writeTestKey(t *testing.T, dir string) string {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = pub
	return path
}

func TestGetPublicKeyAndSign(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)
	t.Setenv(envPrivateKey, keyPath)

	signer, err := loadPrivateKey()
	if err != nil {
		t.Fatalf("loadPrivateKey: %v", err)
	}

	expected := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))

	got, err := GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey error: %v", err)
	}
	if got != expected {
		t.Errorf("public key mismatch\nexpected: %s\n   got: %s", expected, got)
	}

	msg := []byte("hello")
	sig, err := Sign(context.Background(), msg)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	parts := strings.SplitN(sig, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("bad signature format: %s", sig)
	}
	if parts[0] != strings.TrimPrefix(expected, signer.PublicKey().Type()+" ") {
		t.Errorf("public key prefix mismatch")
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	if err := signer.PublicKey().Verify(msg, &ssh.Signature{Format: signer.PublicKey().Type(), Blob: data}); err != nil {
		t.Errorf("signature verify failed: %v", err)
	}
}
