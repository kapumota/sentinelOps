package access

import "strings"

func CanViewAllTunnels(role string) bool {
	switch normalize(role) {
	case "teacher", "auditor", "admin":
		return true
	default:
		return false
	}
}

func CanCloseTunnel(role, requester, owner string) bool {
	if normalize(role) == "admin" {
		return true
	}
	return normalize(requester) != "" && normalize(requester) == normalize(owner)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
