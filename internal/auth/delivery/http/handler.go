// Package http provides Chi handlers for the Auth REST surface.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/noggrj/fiapx-auth-service/internal/auth/domain"
	"github.com/noggrj/fiapx-auth-service/internal/auth/usecase"
	"github.com/noggrj/fiapx-auth-service/internal/platform/httpauth"
	"github.com/noggrj/fiapx-auth-service/internal/platform/jwt"
)

// Handler exposes the Auth REST surface.
type Handler struct {
	register *usecase.RegisterUseCase
	login    *usecase.LoginUseCase
	verifier *jwt.Verifier
	log      *slog.Logger
}

func NewHandler(reg *usecase.RegisterUseCase, lg *usecase.LoginUseCase, v *jwt.Verifier, log *slog.Logger) *Handler {
	return &Handler{register: reg, login: lg, verifier: v, log: log}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/auth/register", h.HandleRegister)
	r.Post("/auth/login", h.HandleLogin)
	r.With(httpauth.Middleware(h.verifier)).Get("/auth/me", h.HandleMe)
}

// ---------- POST /auth/register ----------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	u, err := h.register.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyRegistered) {
			writeErr(w, http.StatusConflict, err)
			return
		}
		writeErr(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, registerResponse{ID: u.ID.String(), Email: u.Email})
}

// ---------- POST /auth/login ----------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	token, err := h.login.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, err)
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// ---------- GET /auth/me (protected) ----------

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"userId": httpauth.UserID(r.Context()),
		"email":  httpauth.Email(r.Context()),
	})
}

// ---------- helpers ----------

func decodeBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
