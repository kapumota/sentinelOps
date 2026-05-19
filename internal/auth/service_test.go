package auth

import "testing"

func TestAuthenticate(t *testing.T) {
	svc := NewDefaultService()

	t.Run("valid student", func(t *testing.T) {
		identity, err := svc.Authenticate("student", "student123!")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if identity.Username != "student" {
			t.Fatalf("expected username student, got %s", identity.Username)
		}
		if identity.Role != RoleStudent {
			t.Fatalf("expected role %s, got %s", RoleStudent, identity.Role)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		_, err := svc.Authenticate("student", "wrong")
		if err == nil {
			t.Fatal("expected authentication error, got nil")
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		_, err := svc.Authenticate("ghost", "ghost123!")
		if err == nil {
			t.Fatal("expected authentication error, got nil")
		}
	})
}

func TestAuthenticateEnvOverride(t *testing.T) {
	t.Setenv("APP_AUTH_ADMIN_PASSWORD", "admin-secret")
	svc := NewDefaultService()

	identity, err := svc.Authenticate("admin", "admin-secret")
	if err != nil {
		t.Fatalf("expected admin env password to authenticate: %v", err)
	}
	if identity.Role != RoleAdmin {
		t.Fatalf("expected admin role, got %s", identity.Role)
	}
}

func TestAuthenticateDoesNotTrimPassword(t *testing.T) {
	t.Setenv("APP_AUTH_STUDENT_PASSWORD", " student-secret ")
	svc := NewDefaultService()

	if _, err := svc.Authenticate("student", "student-secret"); err == nil {
		t.Fatal("expected password without surrounding spaces to fail")
	}
	if _, err := svc.Authenticate("student", " student-secret "); err != nil {
		t.Fatalf("expected exact password to authenticate: %v", err)
	}
}
