package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type Router interface {
	Route(ctx context.Context, userID string, message json.RawMessage) (string, error)
}

type Handler struct {
	router Router
}

func NewHandler(router Router) *Handler { return &Handler{router: router} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/route", h.handleRoute)
}

type RouteRequest struct {
	UserID  string          `json:"user_id"`
	Message json.RawMessage `json:"message"`
}

type RouteResponse struct {
	Status            string `json:"status"`
	GatewayInstanceID string `json:"gateway_instance_id,omitempty"`
}

func (h *Handler) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	if len(req.Message) == 0 {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	gatewayInstanceID, err := h.router.Route(r.Context(), req.UserID, req.Message)
	if err != nil {
		if errors.Is(err, ErrOffline) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "offline"})
			return
		}
		http.Error(w, "failed to route message", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, RouteResponse{Status: "queued", GatewayInstanceID: gatewayInstanceID})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
