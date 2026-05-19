package commands

import (
	"context"
	"fmt"
	"time"
)

type WhoAmICommand struct{}

func NewWhoAmICommand() Command {
	return WhoAmICommand{}
}

func (WhoAmICommand) Name() string {
	return "whoami"
}

func (WhoAmICommand) Aliases() []string {
	return []string{"me", "identity"}
}

func (WhoAmICommand) Description() string {
	return "Muestra la identidad autenticada y el rol de la sesión"
}

func (WhoAmICommand) Usage() string {
	return "whoami"
}

func (WhoAmICommand) Execute(_ context.Context, rt Runtime, _ []string) (string, error) {
	authAt := "not_authenticated"
	if rt.Session.IsAuthenticated() {
		authAt = rt.Session.AuthenticatedAt.Format(time.RFC3339)
	}

	return fmt.Sprintf(
		"username=%s\nrole=%s\nauthenticated=%t\nauthenticated_at=%s\nsession_id=%s",
		rt.Session.Username,
		rt.Session.Role,
		rt.Session.IsAuthenticated(),
		authAt,
		rt.Session.ID,
	), nil
}
