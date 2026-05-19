package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	policypkg "sentinelops/internal/policy"
)

type PolicyCommand struct {
	policyService *policypkg.Service
}

func NewPolicyCommand(policyService *policypkg.Service) Command {
	return PolicyCommand{policyService: policyService}
}

func (c PolicyCommand) Name() string {
	return "policy"
}

func (c PolicyCommand) Aliases() []string {
	return []string{"policies"}
}

func (c PolicyCommand) Description() string {
	return "Evalúa reglas Rego reales sobre el perfil activo"
}

func (c PolicyCommand) Usage() string {
	return "policy [text|json]"
}

func (c PolicyCommand) Execute(_ context.Context, rt Runtime, args []string) (string, error) {
	result := c.policyService.Check(rt.Config.Profile)

	if len(args) > 0 && strings.EqualFold(args[0], "json") {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "policy_status=%s\n", result.Status)
	fmt.Fprintf(&b, "profile=%s\n", result.Profile)
	fmt.Fprintf(&b, "evaluated_rules=%d\n", result.EvaluatedRules)

	if len(result.Denies) == 0 {
		b.WriteString("denies=none\n")
	} else {
		b.WriteString("denies:\n")
		for _, deny := range result.Denies {
			fmt.Fprintf(&b, "- %s\n", deny)
		}
	}

	if len(result.Warnings) == 0 {
		b.WriteString("warnings=none")
	} else {
		b.WriteString("warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}
