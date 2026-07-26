package settings

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/artumont/dotslashstream/internal/auth"
	"github.com/artumont/dotslashstream/internal/httpx"
)

// ── Handler Implementation ───────────────────────────────────────────────────

// Handler serves admin-only global settings routes.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
}

func NewHandler(svc *Service, authSvc *auth.Service) *Handler {
	return &Handler{svc: svc, authSvc: authSvc}
}

// RegisterRoutes mounts admin-only settings endpoints.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	adminOnly := func(next http.HandlerFunc) http.Handler {
		return auth.AuthRequired(h.authSvc, auth.AdminRequired(next))
	}

	mux.Handle("GET /settings", adminOnly(h.get))
	mux.Handle("PATCH /settings", adminOnly(h.update))
}

// ── Route Implementation ─────────────────────────────────────────────────────

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	values, err := h.svc.Get(r.Context())
	if err != nil {
		log.Printf("Get settings error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, values)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var request UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	values, err := h.svc.Update(r.Context(), request)
	if errors.Is(err, ErrNoSettingsUpdated) {
		httpx.WriteError(w, http.StatusBadRequest, "no settings supplied")
		return
	}
	if err != nil {
		log.Printf("Update settings error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, values)
}
