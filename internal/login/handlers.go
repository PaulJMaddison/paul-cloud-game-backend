package login

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		generatedID, err := newUUID()
		if err != nil {
			http.Error(w, "could not create correlation id", http.StatusInternalServerError)
			return
		}
		correlationID = generatedID
	}
	resp, err := h.svc.Login(r.Context(), req, correlationID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		}
		http.Error(w, err.Error(), status)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	userID, _, err := h.svc.ParseToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, MeResponse{User: user})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
