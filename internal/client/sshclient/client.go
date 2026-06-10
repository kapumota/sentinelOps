package sshclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"sentinelops/internal/client/knownhostscb"
	"sentinelops/internal/security"
)

type Config struct {
	Addr              string
	User              string
	Password          string
	IdentityFile      string
	KnownHostsPath    string
	StrictHostKey     bool
	AcceptUnknownHost bool
	Command           string
}

func Run(cfg Config) error {
	if strings.TrimSpace(cfg.Addr) == "" {
		return fmt.Errorf("la dirección del cliente SSH está vacía")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return fmt.Errorf("el usuario del cliente SSH está vacío")
	}

	hostKeyCallback, err := knownhostscb.New(cfg.KnownHostsPath, cfg.StrictHostKey, cfg.AcceptUnknownHost)
	if err != nil {
		return err
	}

	authMethods := make([]ssh.AuthMethod, 0, 2)
	if strings.TrimSpace(cfg.IdentityFile) != "" {
		signer, err := loadSigner(cfg.IdentityFile)
		if err != nil {
			return err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}
	if len(authMethods) == 0 {
		return fmt.Errorf("no hay un método de autenticación configurado; proporciona contraseña o archivo de identidad")
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", cfg.Addr, clientConfig)
	if err != nil {
		return fmt.Errorf("falló la conexión SSH: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("crear sesión SSH: %w", err)
	}
	defer session.Close()

	if strings.TrimSpace(cfg.Command) != "" {
		output, err := session.CombinedOutput(cfg.Command)
		if len(output) > 0 {
			fmt.Print(string(output))
		}
		if err != nil {
			return fmt.Errorf("falló la ejecución remota: %w", err)
		}
		return nil
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if err := session.Shell(); err != nil {
		return fmt.Errorf("falló el inicio de la shell: %w", err)
	}

	return session.Wait()
}

func loadSigner(path string) (ssh.Signer, error) {
	safePath, err := security.ValidateFilesystemPath(path, "clave privada SSH")
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- safePath fue normalizada antes de leer la clave privada.
	raw, err := os.ReadFile(safePath)
	if err != nil {
		return nil, fmt.Errorf("leer clave privada %s: %w", safePath, err)
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("interpretar clave privada %s: %w", safePath, err)
	}
	return signer, nil
}
