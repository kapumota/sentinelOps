package auth

import "testing"

func clearAuthEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"LAB_PASSWORD_STUDENT",
		"LAB_PASSWORD_TEACHER",
		"LAB_PASSWORD_AUDITOR",
		"LAB_PASSWORD_ADMIN",
		"APP_AUTH_STUDENT_PASSWORD",
		"APP_AUTH_TEACHER_PASSWORD",
		"APP_AUTH_AUDITOR_PASSWORD",
		"APP_AUTH_ADMIN_PASSWORD",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestAuthenticate(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("LAB_PASSWORD_STUDENT", "student-secret")
	svc := NewDefaultService()

	t.Run("valid student", func(t *testing.T) {
		identity, err := svc.Authenticate("student", "student-secret")
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
		_, err := svc.Authenticate("ghost", "ghost-secret")
		if err == nil {
			t.Fatal("expected authentication error, got nil")
		}
	})
}

func TestAuthenticateEnvOverride(t *testing.T) {
	clearAuthEnv(t)
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

func TestAuthenticateLabPasswordTakesPrecedence(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("LAB_PASSWORD_STUDENT", "student-lab-secret")
	t.Setenv("APP_AUTH_STUDENT_PASSWORD", "student-legacy-secret")
	svc := NewDefaultService()

	if _, err := svc.Authenticate("student", "student-lab-secret"); err != nil {
		t.Fatalf("expected lab password to authenticate: %v", err)
	}
	if _, err := svc.Authenticate("student", "student-legacy-secret"); err == nil {
		t.Fatal("expected legacy password to fail when lab password is present")
	}
}

func TestAuthenticateGeneratesRandomPasswordWhenMissing(t *testing.T) {
	clearAuthEnv(t)
	svc1 := NewDefaultService()
	svc2 := NewDefaultService()

	if svc1.users["student"].password == "" {
		t.Fatal("expected generated student password")
	}
	if svc1.users["student"].password == svc2.users["student"].password {
		t.Fatal("expected generated passwords to differ")
	}
}

func TestAuthenticateDoesNotTrimPassword(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("APP_AUTH_STUDENT_PASSWORD", " student-secret ")
	svc := NewDefaultService()

	if _, err := svc.Authenticate("student", "student-secret"); err == nil {
		t.Fatal("expected password without surrounding spaces to fail")
	}
	if _, err := svc.Authenticate("student", " student-secret "); err != nil {
		t.Fatalf("expected exact password to authenticate: %v", err)
	}
}
