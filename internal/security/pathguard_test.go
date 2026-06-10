package security

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePathInsideBaseRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	_, err := ValidatePathInsideBase(base, filepath.Join("..", "secreto"), "archivo")
	if err == nil {
		t.Fatal("se esperaba error para ruta fuera del directorio base")
	}
}

func TestValidatePathInsideBaseAcceptsRelativePath(t *testing.T) {
	base := t.TempDir()
	got, err := ValidatePathInsideBase(base, "known_hosts", "archivo")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.HasPrefix(got, base) {
		t.Fatalf("ruta fuera del directorio base: %s", got)
	}
}
