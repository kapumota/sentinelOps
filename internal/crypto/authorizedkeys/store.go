package authorizedkeys

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: strings.TrimSpace(dir)}
}

func (s *Store) IsAuthorized(username string, key ssh.PublicKey) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("authorized keys store is nil")
	}
	if key == nil {
		return false, fmt.Errorf("public key is nil")
	}

	path, err := s.pathForUser(username)
	if err != nil {
		return false, err
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read authorized keys for %s: %w", username, err)
	}

	remaining := raw
	for len(bytes.TrimSpace(remaining)) > 0 {
		parsed, _, _, rest, err := ssh.ParseAuthorizedKey(remaining)
		if err != nil {
			return false, fmt.Errorf("parse authorized keys for %s: %w", username, err)
		}
		if bytes.Equal(parsed.Marshal(), key.Marshal()) {
			return true, nil
		}
		remaining = rest
	}

	return false, nil
}

func (s *Store) pathForUser(username string) (string, error) {
	if s.dir == "" {
		return "", fmt.Errorf("authorized keys directory is empty")
	}
	user := strings.TrimSpace(username)
	if user == "" {
		return "", fmt.Errorf("username is empty")
	}
	if user != filepath.Base(user) || strings.Contains(user, string(filepath.Separator)) {
		return "", fmt.Errorf("invalid username for authorized keys path")
	}
	return filepath.Join(s.dir, user), nil
}
