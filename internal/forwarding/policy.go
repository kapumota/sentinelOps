package forwarding

import (
	"net"
	"sort"
	"strconv"
	"strings"
)

type Policy struct {
	localEnabled       bool
	localAllowlist     map[string]struct{}
	localAllowedRoles  map[string]struct{}
	remoteEnabled      bool
	remoteBindAllow    map[string]struct{}
	remoteAllowedRoles map[string]struct{}
}

func NewPolicy(
	localEnabled bool,
	localAllowlistCSV string,
	localRolesCSV string,
	remoteEnabled bool,
	remoteBindAllowlistCSV string,
	remoteRolesCSV string,
) *Policy {
	return &Policy{
		localEnabled:       localEnabled,
		localAllowlist:     parseCSVSet(localAllowlistCSV),
		localAllowedRoles:  parseCSVSet(localRolesCSV),
		remoteEnabled:      remoteEnabled,
		remoteBindAllow:    parseCSVSet(remoteBindAllowlistCSV),
		remoteAllowedRoles: parseCSVSet(remoteRolesCSV),
	}
}

func (p *Policy) LocalEnabled() bool {
	return p != nil && p.localEnabled
}

func (p *Policy) RemoteEnabled() bool {
	return p != nil && p.remoteEnabled
}

func (p *Policy) AllowLocalTarget(role, host string, port uint32) bool {
	if p == nil || !p.localEnabled {
		return false
	}
	if len(p.localAllowedRoles) > 0 {
		if _, ok := p.localAllowedRoles[normalize(role)]; !ok {
			return false
		}
	}
	_, ok := p.localAllowlist[NormalizeTarget(host, port)]
	return ok
}

func (p *Policy) AllowRemoteBind(role, host string, port uint32) bool {
	if p == nil || !p.remoteEnabled {
		return false
	}
	if len(p.remoteAllowedRoles) > 0 {
		if _, ok := p.remoteAllowedRoles[normalize(role)]; !ok {
			return false
		}
	}
	_, ok := p.remoteBindAllow[NormalizeTarget(host, port)]
	return ok
}

func (p *Policy) AllowedLocalTargets() []string {
	if p == nil {
		return nil
	}
	return sortedKeys(p.localAllowlist)
}
func (p *Policy) AllowedRemoteBinds() []string {
	if p == nil {
		return nil
	}
	return sortedKeys(p.remoteBindAllow)
}
func (p *Policy) AllowedLocalRoles() []string {
	if p == nil {
		return nil
	}
	return sortedKeys(p.localAllowedRoles)
}
func (p *Policy) AllowedRemoteRoles() []string {
	if p == nil {
		return nil
	}
	return sortedKeys(p.remoteAllowedRoles)
}

func NormalizeTarget(host string, port uint32) string {
	return strings.ToLower(net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(int(port))))
}

func parseCSVSet(value string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		v := normalize(item)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func sortedKeys(input map[string]struct{}) []string {
	out := make([]string, 0, len(input))
	for k := range input {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
