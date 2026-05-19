package commands

import (
	"context"
	"time"
)

type DateCommand struct{}

func NewDateCommand() Command {
	return DateCommand{}
}

func (DateCommand) Name() string {
	return "date"
}

func (DateCommand) Aliases() []string {
	return []string{"d", "time"}
}

func (DateCommand) Description() string {
	return "Devuelve fecha y hora del servidor en formato RFC3339"
}

func (DateCommand) Usage() string {
	return "date"
}

func (DateCommand) Execute(_ context.Context, _ Runtime, _ []string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
