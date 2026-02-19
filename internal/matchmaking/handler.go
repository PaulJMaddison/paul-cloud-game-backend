package matchmaking

import (
	"encoding/json"
	"net/http"
	"strings"
)

type TokenParser interface {
	ParseToken(token string) (string, string, error)
}

type Handler struct {
	svc   *Service
	auth  TokenParser
	newID func() (string, error)
}

func NewHandler(svc *Service, auth TokenParser) *Handler {
	return &Handler{svc: svc, auth: auth, newID: newUUID}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/matchmaking/enqueue", h.handleEnqueue)
}

func (h *Handler) handleEnqueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := h.userIDFromAuth(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		id, err := h.newID()
		if err != nil {
			http.Error(w, "could not create correlation id", http.StatusInternalServerError)
			return
		}
		correlationID = id
	}
	if err := h.svc.Enqueue(r.Context(), userID, correlationID); err != nil {
		http.Error(w, "enqueue failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *Handler) userIDFromAuth(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	userID, _, err := h.auth.ParseToken(token)
	if err != nil || userID == "" {
		return "", false
	}
	return userID, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
