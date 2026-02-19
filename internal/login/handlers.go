package login

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginService interface {
	Login(ctx context.Context, req LoginRequest, correlationID string) (LoginResponse, error)
	Me(ctx context.Context, userID string) (UserProfile, error)
	ParseToken(token string) (string, string, error)
}

type Handler struct {
	svc LoginService
}

func NewHandler(svc LoginService) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/login", h.handleLogin)
	mux.HandleFunc("/v1/me", h.handleMe)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	if err := req.Validate(); err != nil {
		apierror.Write(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		generatedID, err := newUUID()
		if err != nil {
			apierror.Write(w, http.StatusInternalServerError, "internal_error", "could not create correlation id")
			return
		}
		correlationID = generatedID
	}
	resp, err := h.svc.Login(r.Context(), req, correlationID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "internal_error"
		if errors.Is(err, ErrInvalidCredentials) {
			status = http.StatusUnauthorized
			code = "invalid_credentials"
		}
		apierror.Write(w, status, code, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	userID, _, err := h.svc.ParseToken(token)
	if err != nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "invalid token")
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, MeResponse{User: user})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
