package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/domain"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/usecase"
)

func TestRegister_HappyPath(t *testing.T) {
	repo := newFakeUserRepo()
	uc := usecase.NewRegister(repo, silentLogger())

	u, err := uc.Execute(context.Background(), "new@user.com", "supersecret123")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if u.Email != "new@user.com" {
		t.Fatalf("unexpected email: %s", u.Email)
	}
	if u.PasswordHash == "" || u.PasswordHash == "supersecret123" {
		t.Fatal("password must be hashed, not stored/echoed as plaintext")
	}

	stored, err := repo.GetByEmail(context.Background(), "new@user.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if stored.ID != u.ID {
		t.Fatal("stored user id mismatch")
	}
}

func TestRegister_RejectsWeakPassword(t *testing.T) {
	repo := newFakeUserRepo()
	uc := usecase.NewRegister(repo, silentLogger())

	if _, err := uc.Execute(context.Background(), "new@user.com", "short"); err == nil {
		t.Fatal("expected error for weak password")
	}
}

func TestRegister_RejectsInvalidEmail(t *testing.T) {
	repo := newFakeUserRepo()
	uc := usecase.NewRegister(repo, silentLogger())

	if _, err := uc.Execute(context.Background(), "not-an-email", "supersecret123"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestRegister_RejectsDuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	uc := usecase.NewRegister(repo, silentLogger())

	if _, err := uc.Execute(context.Background(), "dup@user.com", "supersecret123"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := uc.Execute(context.Background(), "dup@user.com", "anotherpassword")
	if !errors.Is(err, domain.ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}
