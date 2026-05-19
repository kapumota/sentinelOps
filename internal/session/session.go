package session

import (
	"fmt"
	"sync/atomic"
	"time"
)

var sequence atomic.Uint64

type Session struct {
	ID              string
	RemoteAddr      string
	Transport       string
	Authn           string
	ConnectedAt     time.Time
	AuthenticatedAt time.Time
	Username        string
	Role            string
	CommandCount    int
}

func New(remoteAddr string) *Session {
	id := fmt.Sprintf("sess-%06d", sequence.Add(1))

	return &Session{
		ID:          id,
		RemoteAddr:  remoteAddr,
		ConnectedAt: time.Now().UTC(),
	}
}

func (s *Session) SetIdentity(username, role string) {
	s.Username = username
	s.Role = role
	s.AuthenticatedAt = time.Now().UTC()
}

func (s *Session) SetTransport(transport string) {
	s.Transport = transport
}

func (s *Session) SetAuthn(method string) {
	s.Authn = method
}

func (s *Session) IncrementCommands() {
	s.CommandCount++
}

func (s *Session) Uptime() time.Duration {
	return time.Since(s.ConnectedAt)
}

func (s *Session) IsAuthenticated() bool {
	return s.Username != ""
}
