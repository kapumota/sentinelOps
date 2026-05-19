package auth

import (
	"crypto/subtle"
	"errors"
	"os"
	"strings"
)

type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAuditor Role = "auditor"
	RoleAdmin   Role = "admin"
)

type Identity struct {
	Username string
	Role     Role
}

type userRecord struct {
	password string
	role     Role
}

type Authenticator interface {
	Authenticate(username, password string) (Identity, error)
}

type Service struct {
	users map[string]userRecord
}

func NewDefaultService() *Service {
	return &Service{
		users: map[string]userRecord{
			"student": newUserRecord("APP_AUTH_STUDENT_PASSWORD", "student123!", RoleStudent),
			"teacher": newUserRecord("APP_AUTH_TEACHER_PASSWORD", "teacher123!", RoleTeacher),
			"auditor": newUserRecord("APP_AUTH_AUDITOR_PASSWORD", "auditor123!", RoleAuditor),
			"admin":   newUserRecord("APP_AUTH_ADMIN_PASSWORD", "admin123!", RoleAdmin),
		},
	}
}

func newUserRecord(envKey, fallback string, role Role) userRecord {
	secret, ok := os.LookupEnv(envKey)
	if !ok || secret == "" {
		secret = fallback
	}
	return userRecord{password: secret, role: role}
}

func (s *Service) Authenticate(username, password string) (Identity, error) {
	user := strings.TrimSpace(strings.ToLower(username))
	pass := password

	record, ok := s.users[user]
	if !ok {
		return Identity{}, errors.New("invalid credentials")
	}
	if !securePasswordEqual(pass, record.password) {
		return Identity{}, errors.New("invalid credentials")
	}

	return Identity{Username: user, Role: record.role}, nil
}

func securePasswordEqual(actual, expected string) bool {
	if expected == "" {
		return false
	}
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
