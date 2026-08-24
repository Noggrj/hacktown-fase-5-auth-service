package domain_test

import (
	"testing"

	"github.com/noggrj/fiapx-auth-service/internal/auth/domain"
)

func TestNewUser_NormalizesEmail(t *testing.T) {
	u, err := domain.NewUser("  Someone@Example.COM  ")
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if u.Email != "someone@example.com" {
		t.Fatalf("expected normalized email, got %q", u.Email)
	}
	if u.ID.String() == "" {
		t.Fatal("expected a generated ID")
	}
}

func TestNewUser_RejectsInvalidEmail(t *testing.T) {
	cases := []string{"", "not-an-email", "missing-domain@", "@missing-local.com"}
	for _, c := range cases {
		if _, err := domain.NewUser(c); err == nil {
			t.Fatalf("expected error for invalid email %q", c)
		}
	}
}

func TestValidatePassword_RejectsShort(t *testing.T) {
	if err := domain.ValidatePassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestValidatePassword_AcceptsLongEnough(t *testing.T) {
	if err := domain.ValidatePassword("longenough123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
