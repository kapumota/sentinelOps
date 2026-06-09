package secrets

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"
)

func GeneratePassword(length int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		panic(fmt.Errorf("no se pudo generar una contraseña segura: %w", err))
	}
	for i := range buffer {
		buffer[i] = alphabet[int(buffer[i])%len(alphabet)]
	}
	return string(buffer)
}

func LogGeneratedCredential(context, user, password string) {
	separator := strings.Repeat("-", 60)
	log.Printf("\n%s\nCredencial temporal generada para %s\nUsuario: %s\nContraseña: %s\n%s\n", separator, context, user, password, separator)
}
