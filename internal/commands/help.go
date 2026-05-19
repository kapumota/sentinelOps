package commands

import (
	"context"
	"fmt"
	"strings"
)

type HelpCommand struct{}

func NewHelpCommand() Command {
	return HelpCommand{}
}

func (HelpCommand) Name() string {
	return "help"
}

func (HelpCommand) Aliases() []string {
	return []string{"?", "h"}
}

func (HelpCommand) Description() string {
	return "Muestra la lista de comandos disponibles"
}

func (HelpCommand) Usage() string {
	return "help"
}

func (HelpCommand) Execute(_ context.Context, rt Runtime, _ []string) (string, error) {
	var b strings.Builder

	b.WriteString("Comandos de SentinelOps:\n")
	for _, cmd := range rt.Registry.List() {
		fmt.Fprintf(&b, "- %-8s %s\n", cmd.Name(), cmd.Description())
		fmt.Fprintf(&b, "  uso: %s\n", cmd.Usage())
	}

	b.WriteString("- quit     Cierra la sesión\n")
	b.WriteString("  uso: quit")

	return b.String(), nil
}
