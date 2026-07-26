package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/artumont/dotslashstream/internal/httpx"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

const (
	maxUsernameBytes = 64
	maxEmailBytes    = 254
	maxPasswordBytes = 72 - saltLength
)

// ── Handler Implementation ───────────────────────────────────────────────────

// SignupPolicy provides the global policy required for account registration.
type SignupPolicy interface {
	AllowSignupWithoutInvite(ctx context.Context) (bool, error)
}

// Initializer tracks whether the first admin has been created.
type Initializer interface {
	IsInitialized(ctx context.Context) (bool, error)
	MarkInitialized(ctx context.Context) error
}

// Handler implements RouteRegistrar for auth endpoints.
type Handler struct {
	svc          *Service
	signupPolicy SignupPolicy
	initializer  Initializer
}

// NewHandler creates a new auth Handler backed by the given Service and signup policy.
func NewHandler(svc *Service, signupPolicy SignupPolicy, initializer Initializer) *Handler {
	return &Handler{svc: svc, signupPolicy: signupPolicy, initializer: initializer}
}

// RegisterRoutes mounts all auth endpoints on the provided ServeMux.
// Middleware is applied per-route via decorators.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/register/admin", h.registerAdmin)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.HandleFunc("POST /auth/refresh", h.refresh)
	mux.Handle("POST /auth/change-password", AuthRequired(h.svc, http.HandlerFunc(h.changePassword)))
	mux.Handle("POST /auth/invite/generate", AuthRequired(h.svc, AdminRequired(http.HandlerFunc(h.generateInvite))))
}

// ── Route Implementation ─────────────────────────────────────────────────────

// register handles POST /auth/register. It creates a new user account,
// optionally consuming an invite token, and returns a JWT pair on success.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}
	if len(req.Username) > maxUsernameBytes || len(req.Email) > maxEmailBytes || len(req.Password) > maxPasswordBytes {
		httpx.WriteError(w, http.StatusBadRequest, "username, email, or password is too long")
		return
	}
	if !usernameRe.MatchString(req.Username) {
		httpx.WriteError(w, http.StatusBadRequest, "username must be alphanumeric")
		return
	}

	if req.Invite == "" {
		if h.signupPolicy == nil {
			log.Print("Register error: signup policy is not configured")
			httpx.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		allowed, err := h.signupPolicy.AllowSignupWithoutInvite(r.Context())
		if err != nil {
			log.Printf("Register error: read signup policy: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !allowed {
			httpx.WriteError(w, http.StatusForbidden, "an invite is required")
			return
		}
	} else if _, err := h.svc.VerifyInvite(r.Context(), req.Invite); err != nil {
		httpx.WriteError(w, http.StatusForbidden, err.Error())
		return
	}

	_, err := h.svc.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already") {
			httpx.WriteError(w, http.StatusConflict, "username or email already taken")
			return
		}
		log.Printf("Register error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Consume invite after successful registration
	if req.Invite != "" {
		if err := h.svc.ConsumeInvite(r.Context(), req.Invite); err != nil {
			log.Printf("Warning: Failed to consume invite %v", err)
		}
	}

	accessToken, refreshToken, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("Auto-login after registration failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "account created but login failed")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// registerAdmin handles POST /auth/register/admin. It creates the initial admin
// user when the system has not yet been initialized. After the first admin is
// created this endpoint permanently returns 404.
func (h *Handler) registerAdmin(w http.ResponseWriter, r *http.Request) {
	if h.initializer == nil {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	initialized, err := h.initializer.IsInitialized(r.Context())
	if err != nil {
		log.Printf("Register admin error: check init state: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if initialized {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}
	if len(req.Username) > maxUsernameBytes || len(req.Email) > maxEmailBytes || len(req.Password) > maxPasswordBytes {
		httpx.WriteError(w, http.StatusBadRequest, "username, email, or password is too long")
		return
	}
	if !usernameRe.MatchString(req.Username) {
		httpx.WriteError(w, http.StatusBadRequest, "username must be alphanumeric")
		return
	}

	if _, err := h.svc.RegisterAdmin(r.Context(), req.Username, req.Email, req.Password); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already") {
			httpx.WriteError(w, http.StatusConflict, "username or email already taken")
			return
		}
		log.Printf("Register admin error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.initializer.MarkInitialized(r.Context()); err != nil {
		log.Printf("Register admin error: mark initialized: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "account created but setup failed")
		return
	}

	accessToken, refreshToken, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("Auto-login after admin registration failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "account created but login failed")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// login handles POST /auth/login. It validates credentials and returns a JWT pair.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if len(req.Username) > maxUsernameBytes || len(req.Password) > maxPasswordBytes {
		httpx.WriteError(w, http.StatusBadRequest, "username or password is too long")
		return
	}
	if !usernameRe.MatchString(req.Username) {
		httpx.WriteError(w, http.StatusBadRequest, "username must be alphanumeric")
		return
	}

	accessToken, refreshToken, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		log.Printf("Login error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// refresh handles POST /auth/refresh. It exchanges a valid refresh token for a new JWT pair.
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	accessToken, refreshToken, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// changePassword handles POST /auth/change-password. It requires authentication
// and updates the user's password after verifying the old one.
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		httpx.WriteError(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}
	if len(req.OldPassword) > maxPasswordBytes || len(req.NewPassword) > maxPasswordBytes {
		httpx.WriteError(w, http.StatusBadRequest, "old_password or new_password is too long")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), user.ID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "incorrect current password")
			return
		}
		log.Printf("Change password error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// generateInvite handles POST /auth/invite/generate. It requires admin authentication
// and creates a new invite token with the specified TTL and max use count.
func (h *Handler) generateInvite(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r)
	if user == nil {
		return
	}

	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MaxUses < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "max_uses must be at least 1")
		return
	}

	ttl, err := time.ParseDuration(req.TTL)
	if err != nil || ttl <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid ttl (e.g. \"168h\", \"24h\")")
		return
	}

	token, err := h.svc.CreateInvite(r.Context(), user.ID, ttl, req.MaxUses)
	if err != nil {
		log.Printf("Create invite error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, inviteResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(ttl).UTC().Format(time.RFC3339),
		MaxUses:   req.MaxUses,
	})
}
