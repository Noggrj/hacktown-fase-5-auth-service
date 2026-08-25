package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authhttp "github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/delivery/http"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/domain"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/auth/usecase"
	"github.com/noggrj/hacktown-fase-5-auth-service/internal/platform/jwt"
)

const testSecret = "test-secret-at-least-16-bytes"

type fakeRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
}

func newFakeRepo() *fakeRepo { return &fakeRepo{users: map[string]*domain.User{}} }

func (f *fakeRepo) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[u.Email]; ok {
		return domain.ErrEmailAlreadyRegistered
	}
	cp := *u
	f.users[u.Email] = &cp
	return nil
}

func (f *fakeRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func newTestHandler() (*authhttp.Handler, *fakeRepo) {
	repo := newFakeRepo()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	issuer, _ := jwt.NewIssuer(testSecret)
	verifier, _ := jwt.NewVerifier(testSecret)
	h := authhttp.NewHandler(
		usecase.NewRegister(repo, log),
		usecase.NewLogin(repo, issuer, log),
		verifier,
		log,
	)
	return h, repo
}

func newTestRouter() *chi.Mux {
	h, _ := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRegister_HappyPath_Returns201(t *testing.T) {
	r := newTestRouter()
	rec := doJSON(t, r, http.MethodPost, "/auth/register", map[string]string{
		"email": "new@user.com", "password": "supersecret123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegister_DuplicateEmail_Returns409(t *testing.T) {
	r := newTestRouter()
	body := map[string]string{"email": "dup@user.com", "password": "supersecret123"}
	doJSON(t, r, http.MethodPost, "/auth/register", body)
	rec := doJSON(t, r, http.MethodPost, "/auth/register", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestRegister_WeakPassword_Returns422(t *testing.T) {
	r := newTestRouter()
	rec := doJSON(t, r, http.MethodPost, "/auth/register", map[string]string{
		"email": "weak@user.com", "password": "short",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
}

func TestLogin_HappyPath_ReturnsToken(t *testing.T) {
	r := newTestRouter()
	creds := map[string]string{"email": "login@user.com", "password": "supersecret123"}
	if rec := doJSON(t, r, http.MethodPost, "/auth/register", creds); rec.Code != http.StatusCreated {
		t.Fatalf("register setup failed: %d", rec.Code)
	}

	rec := doJSON(t, r, http.MethodPost, "/auth/login", creds)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLogin_WrongPassword_Returns401(t *testing.T) {
	r := newTestRouter()
	doJSON(t, r, http.MethodPost, "/auth/register", map[string]string{"email": "x@user.com", "password": "supersecret123"})
	rec := doJSON(t, r, http.MethodPost, "/auth/login", map[string]string{"email": "x@user.com", "password": "wrongpassword"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMe_WithoutToken_Returns401(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMe_WithValidToken_ReturnsClaims(t *testing.T) {
	r := newTestRouter()
	creds := map[string]string{"email": "me@user.com", "password": "supersecret123"}
	doJSON(t, r, http.MethodPost, "/auth/register", creds)
	loginRec := doJSON(t, r, http.MethodPost, "/auth/login", creds)
	var loginResp map[string]string
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp["token"])
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var meResp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&meResp)
	if meResp["email"] != "me@user.com" {
		t.Fatalf("unexpected /auth/me response: %+v", meResp)
	}
}
