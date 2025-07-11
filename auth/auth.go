package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	// envPrivateKey is the environment variable that can override the path
	// to the SSH private key used for authentication.
	envPrivateKey     = "GOOBLA_PRIVATE_KEY"
	defaultPrivateKey = "id_ed25519"
)

func keyPath() (string, error) {
	if p := os.Getenv(envPrivateKey); p != "" {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".goobla", defaultPrivateKey), nil
}

// loadPrivateKey reads and parses the configured private key. It returns an
// ssh.Signer that can be used for public key retrieval or signing operations.
// The path to the key may be overridden by the GOOBLA_PRIVATE_KEY environment
// variable.
func loadPrivateKey() (ssh.Signer, error) {
	path, err := keyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info(fmt.Sprintf("Failed to load private key: %v", err))
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// GetPublicKey returns the SSH public key corresponding to the configured
// private key. The path can be overridden by the GOOBLA_PRIVATE_KEY environment
// variable.
func GetPublicKey() (string, error) {
	signer, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	publicKey := ssh.MarshalAuthorizedKey(signer.PublicKey())

	return strings.TrimSpace(string(publicKey)), nil
}

// NewNonce returns a base64 encoded nonce of the given length read from r.
func NewNonce(r io.Reader, length int) (string, error) {
	nonce := make([]byte, length)
	if _, err := io.ReadFull(r, nonce); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(nonce), nil
}

// Sign signs the provided bytes with the user's private key and returns a
// string of the form "<public_key>:<signature>". The private key path can be
// overridden by the GOOBLA_PRIVATE_KEY environment variable.
func Sign(ctx context.Context, bts []byte) (string, error) {
	signer, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	// get the pubkey, but remove the type
	publicKey := ssh.MarshalAuthorizedKey(signer.PublicKey())
	parts := bytes.Split(publicKey, []byte(" "))
	if len(parts) < 2 {
		return "", errors.New("malformed public key")
	}

	signedData, err := signer.Sign(rand.Reader, bts)
	if err != nil {
		return "", err
	}

	// signature is <pubkey>:<signature>
	return fmt.Sprintf("%s:%s", bytes.TrimSpace(parts[1]), base64.StdEncoding.EncodeToString(signedData.Blob)), nil
}
