package security

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type InputValidator interface {
	Validate(input string) error
}

type Options struct {
	ExternalEnabled bool
	ExternalBinary  string
	FailOpen        bool
}

type StaticRuleValidator struct {
	maxLength int
	forbidden []string
}

type ExternalProcessValidator struct {
	binary string
}

type ExternalRuntimeError struct {
	binary string
	msg    string
	err    error
}

type HybridValidator struct {
	internal InputValidator
	external InputValidator
	failOpen bool
}

func NewValidator(opts Options) InputValidator {
	internal := NewDefaultValidator()

	if !opts.ExternalEnabled || strings.TrimSpace(opts.ExternalBinary) == "" {
		return internal
	}

	external := &ExternalProcessValidator{binary: strings.TrimSpace(opts.ExternalBinary)}
	return NewHybridValidator(internal, external, opts.FailOpen)
}

func NewDefaultValidator() InputValidator {
	return &StaticRuleValidator{
		maxLength: 120,
		forbidden: []string{"&&", "||", "../", "$(", "`", ";"},
	}
}

func NewHybridValidator(internal, external InputValidator, failOpen bool) InputValidator {
	return &HybridValidator{internal: internal, external: external, failOpen: failOpen}
}

func (v *StaticRuleValidator) Validate(input string) error {
	trimmed := strings.TrimSpace(input)

	if trimmed == "" {
		return errors.New("no se permite una entrada vacía")
	}
	if len(trimmed) > v.maxLength {
		return fmt.Errorf("la entrada supera %d caracteres", v.maxLength)
	}
	for _, token := range v.forbidden {
		if strings.Contains(trimmed, token) {
			return fmt.Errorf("se detectó un token prohibido: %s", token)
		}
	}
	return nil
}

func (v *ExternalProcessValidator) Validate(input string) error {
	cmd := exec.Command(v.binary, input)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = err.Error()
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 10 {
		return errors.New(message)
	}

	return &ExternalRuntimeError{binary: v.binary, msg: message, err: err}
}

func (e *ExternalRuntimeError) Error() string {
	return fmt.Sprintf("fallo de ejecución del validador externo (%s): %s", e.binary, e.msg)
}

func (e *ExternalRuntimeError) Unwrap() error { return e.err }

func (v *HybridValidator) Validate(input string) error {
	if v.internal != nil {
		if err := v.internal.Validate(input); err != nil {
			return err
		}
	}
	if v.external == nil {
		return nil
	}

	err := v.external.Validate(input)
	if err == nil {
		return nil
	}

	var runtimeErr *ExternalRuntimeError
	if errors.As(err, &runtimeErr) {
		if v.failOpen {
			return nil
		}
		return runtimeErr
	}
	return err
}
