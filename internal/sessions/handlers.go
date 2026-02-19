package sessions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/sessions", h.handleCreateSession)
	mux.HandleFunc("/v1/sessions/", h.handleSessionRoutes)
}

func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, correlationID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	session, err := h.svc.CreateSessionForUser(r.Context(), userID, correlationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]Session{"session": session})
}

func (h *Handler) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, correlationID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/sessions/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "assign-server" {
		http.NotFound(w, r)
		return
	}
	resp, err := h.svc.AssignServer(r.Context(), userID, parts[0], correlationID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return "", "", false
	}
	userID, err := h.svc.ParseToken(strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return "", "", false
	}
	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		correlationID, err = newUUID()
		if err != nil {
			http.Error(w, "could not create correlation id", http.StatusInternalServerError)
			return "", "", false
		}
	}
	return userID, correlationID, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
