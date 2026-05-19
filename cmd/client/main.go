package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sentinelops/internal/client/sshclient"
)

func main() {
	home := userHomeDir()

	addr := flag.String("addr", "localhost:2222", "Dirección del servidor SSH")
	user := flag.String("user", "student", "Usuario SSH")
	password := flag.String("password", "", "Autenticación por contraseña")
	identity := flag.String("identity", "", "Ruta de la clave privada para autenticación por clave pública")
	knownHosts := flag.String("known-hosts", filepath.Join(home, ".ssh", "known_hosts"), "Ruta del archivo known_hosts")
	strictHostKey := flag.Bool("strict-host-key", true, "Requerir verificación del host mediante known_hosts")
	acceptUnknown := flag.Bool("accept-unknown-host", true, "Agregar claves de host desconocidas a known_hosts")
	command := flag.String("cmd", "", "Comando remoto único a ejecutar en lugar de la shell interactiva")

	flag.Parse()

	cfg := sshclient.Config{
		Addr:              *addr,
		User:              *user,
		Password:          *password,
		IdentityFile:      *identity,
		KnownHostsPath:    *knownHosts,
		StrictHostKey:     *strictHostKey,
		AcceptUnknownHost: *acceptUnknown,
		Command:           *command,
	}

	if err := sshclient.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "sentinelops-client:", err)
		os.Exit(1)
	}
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
