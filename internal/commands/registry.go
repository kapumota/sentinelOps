package commands

import (
	"context"
	"strings"

	"sentinelops/internal/config"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/session"
)

type Runtime struct {
	Session    *session.Session
	Config     config.Config
	Registry   *Registry
	Forwarding forwarding.Controller
}

type Command interface {
	Name() string
	Aliases() []string
	Description() string
	Usage() string
	Execute(ctx context.Context, rt Runtime, args []string) (string, error)
}

type Registry struct {
	lookup  map[string]Command
	ordered []Command
}

func NewRegistry(cmds ...Command) *Registry {
	r := &Registry{lookup: make(map[string]Command), ordered: make([]Command, 0, len(cmds))}
	seenPrimary := make(map[string]bool)
	for _, cmd := range cmds {
		primary := strings.ToLower(cmd.Name())
		if !seenPrimary[primary] {
			r.ordered = append(r.ordered, cmd)
			seenPrimary[primary] = true
		}
		r.lookup[primary] = cmd
		for _, alias := range cmd.Aliases() {
			r.lookup[strings.ToLower(alias)] = cmd
		}
	}
	return r
}

func (r *Registry) Find(name string) (Command, bool) {
	cmd, ok := r.lookup[strings.ToLower(name)]
	return cmd, ok
}

func (r *Registry) List() []Command {
	out := make([]Command, len(r.ordered))
	copy(out, r.ordered)
	return out
}
