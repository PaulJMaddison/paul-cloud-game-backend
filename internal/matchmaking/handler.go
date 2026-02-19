package matchmaking

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
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
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, ok := h.userIDFromAuth(r)
	if !ok {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		id, err := h.newID()
		if err != nil {
			apierror.Write(w, http.StatusInternalServerError, "internal_error", "could not create correlation id")
			return
		}
		correlationID = id
	}
	if err := h.svc.Enqueue(r.Context(), userID, correlationID); err != nil {
		apierror.Write(w, http.StatusInternalServerError, "internal_error", "enqueue failed")
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
