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

func LogGeneratedCredential(context, user, secretValue string) {
	separator := strings.Repeat("-", 60)
	redacted := RedactSecret(secretValue)
	log.Printf("\n%s\nCredencial temporal generada para %s\nUsuario: %s\nSecreto: %s\n%s\n", separator, context, user, redacted, separator)
}

func RedactSecret(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "<vacío>"
	}
	if len(trimmed) <= 4 {
		return "<redactado>"
	}
	return fmt.Sprintf("<redactado longitud=%d sufijo=%s>", len(trimmed), trimmed[len(trimmed)-4:])
}
