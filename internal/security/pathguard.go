package security

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateFilesystemPath normaliza una ruta antes de usarla en operaciones de archivo.
func ValidateFilesystemPath(path string, field string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("%s está vacío", field)
	}
	if strings.Contains(cleaned, "\x00") {
		return "", fmt.Errorf("%s contiene caracteres no válidos", field)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("normalizar %s: %w", field, err)
	}
	return abs, nil
}

// ValidatePathInsideBase asegura que una ruta quede dentro de un directorio base permitido.
func ValidatePathInsideBase(base string, path string, field string) (string, error) {
	baseAbs, err := ValidateFilesystemPath(base, "directorio base")
	if err != nil {
		return "", err
	}

	candidate := strings.TrimSpace(path)
	if candidate == "" {
		return "", fmt.Errorf("%s está vacío", field)
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(baseAbs, candidate)
	}

	candidateAbs, err := ValidateFilesystemPath(candidate, field)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return "", fmt.Errorf("comparar %s con directorio base: %w", field, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s queda fuera del directorio permitido", field)
	}
	return candidateAbs, nil
}

// ValidateExecutable resuelve un ejecutable sin invocar shell.
func ValidateExecutable(binary string) (string, error) {
	value := strings.TrimSpace(binary)
	if value == "" {
		return "", fmt.Errorf("el ejecutable está vacío")
	}
	if strings.Contains(value, "\x00") {
		return "", fmt.Errorf("el ejecutable contiene caracteres no válidos")
	}
	if strings.ContainsAny(value, `/\\`) {
		return ValidateFilesystemPath(value, "ejecutable")
	}
	resolved, err := exec.LookPath(value)
	if err != nil {
		return "", fmt.Errorf("resolver ejecutable %s: %w", value, err)
	}
	return resolved, nil
}
