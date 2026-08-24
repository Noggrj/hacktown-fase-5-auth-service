package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/noggrj/fiapx-auth-service/internal/auth/domain"
	"github.com/noggrj/fiapx-auth-service/internal/auth/usecase"
	"github.com/noggrj/fiapx-auth-service/internal/platform/jwt"
)

const testSecret = "test-secret-at-least-16-bytes"

func TestLogin_HappyPath_ReturnsVerifiableToken(t *testing.T) {
	repo := newFakeUserRepo()
	registerUC := usecase.NewRegister(repo, silentLogger())
	if _, err := registerUC.Execute(context.Background(), "user@login.com", "supersecret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	issuer, _ := jwt.NewIssuer(testSecret)
	verifier, _ := jwt.NewVerifier(testSecret)
	loginUC := usecase.NewLogin(repo, issuer, silentLogger())

	token, err := loginUC.Execute(context.Background(), "user@login.com", "supersecret123")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	claims, err := verifier.Verify(token)
	if err != nil {
		t.Fatalf("issued token must verify: %v", err)
	}
	if claims.Email != "user@login.com" {
		t.Fatalf("unexpected claims email: %s", claims.Email)
	}
}

func TestLogin_RejectsUnknownEmail(t *testing.T) {
	repo := newFakeUserRepo()
	issuer, _ := jwt.NewIssuer(testSecret)
	loginUC := usecase.NewLogin(repo, issuer, silentLogger())

	_, err := loginUC.Execute(context.Background(), "ghost@user.com", "whatever123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_RejectsWrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	registerUC := usecase.NewRegister(repo, silentLogger())
	_, _ = registerUC.Execute(context.Background(), "user@login.com", "correctpassword")

	issuer, _ := jwt.NewIssuer(testSecret)
	loginUC := usecase.NewLogin(repo, issuer, silentLogger())

	_, err := loginUC.Execute(context.Background(), "user@login.com", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
