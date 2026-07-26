package settings

// Values is the public global-settings document.
type Values struct {
	AllowSignupWithoutInvite bool `json:"allow_signup_without_invite"`
	FirstInitCompleted       bool `json:"first_init_completed"`
}

// UpdateRequest contains mutable settings. Nil means field was omitted.
type UpdateRequest struct {
	AllowSignupWithoutInvite *bool `json:"allow_signup_without_invite"`
}
