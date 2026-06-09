package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"strings"

	"sentinelops/internal/secrets"
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
			"student": newUserRecord("student", RoleStudent, "LAB_PASSWORD_STUDENT", "APP_AUTH_STUDENT_PASSWORD"),
			"teacher": newUserRecord("teacher", RoleTeacher, "LAB_PASSWORD_TEACHER", "APP_AUTH_TEACHER_PASSWORD"),
			"auditor": newUserRecord("auditor", RoleAuditor, "LAB_PASSWORD_AUDITOR", "APP_AUTH_AUDITOR_PASSWORD"),
			"admin":   newUserRecord("admin", RoleAdmin, "LAB_PASSWORD_ADMIN", "APP_AUTH_ADMIN_PASSWORD"),
		},
	}
}

func newUserRecord(username string, role Role, envKeys ...string) userRecord {
	for _, envKey := range envKeys {
		if secret, ok := os.LookupEnv(envKey); ok && secret != "" {
			return userRecord{password: secret, role: role}
		}
	}

	secret := secrets.GeneratePassword(20)
	secrets.LogGeneratedCredential(fmt.Sprintf("usuario de laboratorio %s", username), username, secret)
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
