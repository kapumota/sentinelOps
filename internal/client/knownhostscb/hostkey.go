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

	"sentinelops/internal/security"
)

func New(path string, strict bool, acceptUnknown bool) (ssh.HostKeyCallback, error) {
	if !strict {
		// #nosec G106 -- modo no estricto solo se permite cuando la configuración lo solicita explícitamente para laboratorio.
		return ssh.InsecureIgnoreHostKey(), nil
	}

	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("la ruta de known_hosts está vacía")
	}
	safePath, err := security.ValidateFilesystemPath(path, "known_hosts")
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(safePath), 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio known_hosts: %w", err)
	}

	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		// #nosec G304 -- safePath fue normalizada antes de crear el archivo known_hosts.
		if err := os.WriteFile(safePath, []byte{}, 0o600); err != nil {
			return nil, fmt.Errorf("crear archivo known_hosts: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("consultar archivo known_hosts: %w", err)
	}

	base, err := knownhosts.New(safePath)
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

			// #nosec G304 -- safePath fue normalizada antes de abrir known_hosts para anexar.
			f, openErr := os.OpenFile(safePath, os.O_APPEND|os.O_WRONLY, 0o600)
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
