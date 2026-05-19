package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sentinelops/internal/access"
	"sentinelops/internal/forwarding"
)

type TunnelsCommand struct{}

func NewTunnelsCommand() Command           { return TunnelsCommand{} }
func (TunnelsCommand) Name() string        { return "tunnels" }
func (TunnelsCommand) Aliases() []string   { return []string{"forwards", "forwardlist"} }
func (TunnelsCommand) Description() string { return "Muestra o administra túneles SSH activos" }
func (TunnelsCommand) Usage() string       { return "tunnels [mine|all|close <id>]" }

func (TunnelsCommand) Execute(_ context.Context, rt Runtime, args []string) (string, error) {
	if rt.Forwarding == nil {
		return "active_tunnels=0\nforwarding=disabled", nil
	}
	if len(args) == 0 {
		if access.CanViewAllTunnels(rt.Session.Role) {
			return renderTunnels(rt.Forwarding.Snapshot()), nil
		}
		return renderTunnels(rt.Forwarding.SnapshotByUsername(rt.Session.Username)), nil
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "mine":
		return renderTunnels(rt.Forwarding.SnapshotByUsername(rt.Session.Username)), nil
	case "all":
		if !access.CanViewAllTunnels(rt.Session.Role) {
			return "Acceso denegado: tu rol no puede ver todos los túneles.", nil
		}
		return renderTunnels(rt.Forwarding.Snapshot()), nil
	case "close":
		if len(args) < 2 {
			return "Uso: tunnels close <id>", nil
		}
		id := strings.TrimSpace(args[1])
		tunnel, ok := rt.Forwarding.Get(id)
		if !ok {
			return "Túnel no encontrado: " + id, nil
		}
		if !access.CanCloseTunnel(rt.Session.Role, rt.Session.Username, tunnel.Username) {
			return "Acceso denegado: no puedes cerrar este túnel.", nil
		}
		if !rt.Forwarding.Close(id) {
			return "No se pudo cerrar el túnel: " + id, nil
		}
		return "Túnel cerrado: " + id, nil
	default:
		return "Uso: tunnels [mine|all|close <id>]", nil
	}
}

func renderTunnels(items []forwarding.Tunnel) string {
	if len(items) == 0 {
		return "active_tunnels=0"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "active_tunnels=%d\n", len(items))
	for _, t := range items {
		fmt.Fprintf(&b, "- id=%s direction=%s session_id=%s username=%s bind=%s target=%s origin=%s started_at=%s uptime=%s\n", t.ID, t.Direction, t.SessionID, t.Username, t.Bind, t.Target, t.Origin, t.StartedAt.Format(time.RFC3339), time.Since(t.StartedAt).Round(time.Second))
	}
	return strings.TrimRight(b.String(), "\n")
}
