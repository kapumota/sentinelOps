package knownhostscb

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func New(path string, strict bool, acceptUnknown bool) (ssh.HostKeyCallback, error) {
	if !strict {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("la ruta de known_hosts está vacía")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio known_hosts: %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return nil, fmt.Errorf("crear archivo known_hosts: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("consultar archivo known_hosts: %w", err)
	}

	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("cargar callback de known_hosts: %w", err)
	}

	if !acceptUnknown {
		return base, nil
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := base(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			line := knownhosts.Line([]string{hostname}, key)

			f, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
			if openErr != nil {
				return fmt.Errorf("abrir known_hosts para anexar: %w", openErr)
			}
			defer f.Close()

			if _, writeErr := fmt.Fprintln(f, line); writeErr != nil {
				return fmt.Errorf("agregar entrada a known_hosts: %w", writeErr)
			}

			return nil
		}

		return err
	}, nil
}
