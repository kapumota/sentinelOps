package commands

import (
	"context"
	"fmt"
	"time"
)

type StatusCommand struct{}

func NewStatusCommand() Command {
	return StatusCommand{}
}

func (StatusCommand) Name() string {
	return "status"
}

func (StatusCommand) Aliases() []string {
	return []string{"info"}
}

func (StatusCommand) Description() string {
	return "Muestra estado de la sesión y del perfil activo"
}

func (StatusCommand) Usage() string {
	return "status"
}

func (StatusCommand) Execute(_ context.Context, rt Runtime, _ []string) (string, error) {
	return fmt.Sprintf(
		"app=%s\nenvironment=%s\nprofile=%s\nsession_id=%s\nremote_addr=%s\nconnected_at=%s\nuptime=%s\ncommands_executed=%d",
		rt.Config.AppName,
		rt.Config.Environment,
		rt.Config.Profile,
		rt.Session.ID,
		rt.Session.RemoteAddr,
		rt.Session.ConnectedAt.Format(time.RFC3339),
		rt.Session.Uptime().Round(time.Second),
		rt.Session.CommandCount,
	), nil
}
