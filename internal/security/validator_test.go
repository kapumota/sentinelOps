package security

import (
	"errors"
	"strings"
	"testing"
)

type mockValidator struct {
	err error
}

func (m mockValidator) Validate(_ string) error {
	return m.err
}

func TestStaticRuleValidator(t *testing.T) {
	v := NewDefaultValidator()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid command",
			input:     "status",
			wantError: false,
		},
		{
			name:      "empty command",
			input:     "   ",
			wantError: true,
		},
		{
			name:      "forbidden token",
			input:     "status && whoami",
			wantError: true,
		},
		{
			name:      "too long",
			input:     strings.Repeat("a", 121),
			wantError: true,
		},
	}

	for _, tt := range tests {
		err := v.Validate(tt.input)
		if tt.wantError && err == nil {
			t.Fatalf("%s: expected error, got nil", tt.name)
		}
		if !tt.wantError && err != nil {
			t.Fatalf("%s: expected nil error, got %v", tt.name, err)
		}
	}
}

func TestHybridValidatorRejectsInternalFailure(t *testing.T) {
	v := NewHybridValidator(
		mockValidator{err: errors.New("internal reject")},
		mockValidator{err: nil},
		false,
	)

	err := v.Validate("status")
	if err == nil || err.Error() != "internal reject" {
		t.Fatalf("expected internal reject, got %v", err)
	}
}

func TestHybridValidatorRejectsExternalValidationFailure(t *testing.T) {
	v := NewHybridValidator(
		mockValidator{err: nil},
		mockValidator{err: errors.New("external reject")},
		false,
	)

	err := v.Validate("status")
	if err == nil || err.Error() != "external reject" {
		t.Fatalf("expected external reject, got %v", err)
	}
}

func TestHybridValidatorFailOpenOnRuntimeError(t *testing.T) {
	v := NewHybridValidator(
		mockValidator{err: nil},
		mockValidator{err: &ExternalRuntimeError{
			binary: "input-guard",
			msg:    "binary missing",
			err:    errors.New("exec failed"),
		}},
		true,
	)

	err := v.Validate("status")
	if err != nil {
		t.Fatalf("expected nil error in fail-open mode, got %v", err)
	}
}

func TestHybridValidatorFailClosedOnRuntimeError(t *testing.T) {
	v := NewHybridValidator(
		mockValidator{err: nil},
		mockValidator{err: &ExternalRuntimeError{
			binary: "input-guard",
			msg:    "binary missing",
			err:    errors.New("exec failed"),
		}},
		false,
	)

	err := v.Validate("status")
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
}
