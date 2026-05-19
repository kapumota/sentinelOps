package policy

import (
	"fmt"
	"time"
)

type Result struct {
	Status         string    `json:"status"`
	Profile        string    `json:"profile"`
	EvaluatedRules int       `json:"evaluated_rules"`
	Denies         []string  `json:"denies"`
	Warnings       []string  `json:"warnings"`
	Timestamp      time.Time `json:"timestamp"`
}

type Service struct {
	externalRunner ExternalRunner
}

func NewService(externalRunner ExternalRunner) *Service {
	return &Service{externalRunner: externalRunner}
}

func (s *Service) Check(profile string) Result {
	result := Result{
		Status:         "pass",
		Profile:        profile,
		EvaluatedRules: 4,
		Denies:         []string{},
		Warnings:       []string{},
		Timestamp:      time.Now().UTC(),
	}

	if s.externalRunner == nil {
		result.Status = "warn"
		result.Warnings = append(result.Warnings, "OPA runner no configurado; no se evaluaron políticas externas")
		return result
	}

	input := BuildDeploymentInput(profile)

	denies, warnings, err := s.externalRunner.Check(input)
	if err != nil {
		result.Status = "warn"
		result.Warnings = append(result.Warnings, fmt.Sprintf("No se pudo evaluar Rego/OPA: %v", err))
		return result
	}

	result.Denies = append(result.Denies, denies...)
	result.Warnings = append(result.Warnings, warnings...)

	if len(result.Denies) > 0 {
		result.Status = "fail"
	} else if len(result.Warnings) > 0 {
		result.Status = "warn"
	}

	return result
}
