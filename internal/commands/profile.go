package commands

import (
	"context"
	"fmt"
)

type ProfileCommand struct{}

func NewProfileCommand() Command {
	return ProfileCommand{}
}

func (ProfileCommand) Name() string {
	return "profile"
}

func (ProfileCommand) Aliases() []string {
	return []string{"mode"}
}

func (ProfileCommand) Description() string {
	return "Muestra el perfil operativo actual del laboratorio"
}

func (ProfileCommand) Usage() string {
	return "profile"
}

func (ProfileCommand) Execute(_ context.Context, rt Runtime, _ []string) (string, error) {
	return fmt.Sprintf("active_profile=%s", rt.Config.Profile), nil
}
