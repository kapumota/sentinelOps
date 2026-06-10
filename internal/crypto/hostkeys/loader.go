package hostkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"sentinelops/internal/security"
)

func LoadOrCreateSigner(path string) (ssh.Signer, error) {
	if path == "" {
		return nil, fmt.Errorf("ssh host key path is empty")
	}
	safePath, err := security.ValidateFilesystemPath(path, "ssh host key")
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(safePath), 0o700); err != nil {
		return nil, fmt.Errorf("create host key directory: %w", err)
	}

	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		if err := generateEd25519Key(safePath); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat host key: %w", err)
	}

	// #nosec G304 -- safePath fue normalizada antes de leer la clave host.
	raw, err := os.ReadFile(safePath)
	if err != nil {
		return nil, fmt.Errorf("read host key: %w", err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM in host key file")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS8 host key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("build ssh signer: %w", err)
	}

	return signer, nil
}

func generateEd25519Key(path string) error {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ed25519 host key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal PKCS8 host key: %w", err)
	}

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}

	// #nosec G304 -- path fue normalizada por LoadOrCreateSigner antes de crear la clave.
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("write host key: %w", err)
	}

	return nil
}
