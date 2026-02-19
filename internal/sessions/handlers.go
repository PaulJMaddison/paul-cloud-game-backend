package sessions

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

type Handler struct {
	svc        *Service
	adminToken string
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, adminToken: os.Getenv("ADMIN_TOKEN")}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/sessions", h.handleCreateSession)
	mux.HandleFunc("/v1/sessions/", h.handleSessionRoutes)
	mux.HandleFunc("/admin/v1/users", h.handleAdminUsers)
	mux.HandleFunc("/admin/v1/sessions", h.handleAdminSessions)
	mux.HandleFunc("/admin/v1/broadcast", h.handleAdminBroadcast)
}

func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, correlationID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	session, err := h.svc.CreateSessionForUser(r.Context(), userID, correlationID)
	if err != nil {
		apierror.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]Session{"session": session})
}

func (h *Handler) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
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
			apierror.Write(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		apierror.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.adminAuth(w, r) {
		return
	}
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		apierror.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]User{"users": users})
}

func (h *Handler) handleAdminSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.adminAuth(w, r) {
		return
	}
	sessions, err := h.svc.ListSessions(r.Context())
	if err != nil {
		apierror.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]Session{"sessions": sessions})
}

type adminBroadcastRequest struct {
	Message json.RawMessage `json:"message"`
}

func (h *Handler) handleAdminBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.adminAuth(w, r) {
		return
	}
	var req adminBroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if len(req.Message) == 0 {
		apierror.Write(w, http.StatusBadRequest, "validation_failed", "message is required")
		return
	}
	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		var err error
		correlationID, err = newUUID()
		if err != nil {
			apierror.Write(w, http.StatusInternalServerError, "internal_error", "could not create correlation id")
			return
		}
	}
	count, err := h.svc.BroadcastToOnlineUsers(r.Context(), correlationID, req.Message)
	if err != nil {
		apierror.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"published": count})
}

func (h *Handler) adminAuth(w http.ResponseWriter, r *http.Request) bool {
	token := r.Header.Get("X-Admin-Token")
	if token == "" || h.adminToken == "" || token != h.adminToken {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return false
	}
	return true
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return "", "", false
	}
	userID, err := h.svc.ParseToken(strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "invalid token")
		return "", "", false
	}
	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		correlationID, err = newUUID()
		if err != nil {
			apierror.Write(w, http.StatusInternalServerError, "internal_error", "could not create correlation id")
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
