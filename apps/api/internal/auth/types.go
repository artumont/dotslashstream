package auth

import "github.com/google/uuid"

// ── Requests ─────────────────────────────────────────────────────────────────

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Invite   string `json:"invite,omitempty"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type inviteRequest struct {
	TTL     string `json:"ttl"` // e.g. "168h", "24h"
	MaxUses int    `json:"max_uses"`
}

// ── Responses ────────────────────────────────────────────────────────────────

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsAdmin  bool      `json:"is_admin"`
}

type inviteResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	MaxUses   int    `json:"max_uses"`
}
