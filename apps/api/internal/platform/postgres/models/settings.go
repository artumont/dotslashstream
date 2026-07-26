package models

import (
	"time"

	"github.com/uptrace/bun"
)

// SettingsSingletonID identifies the only global-settings row.
const SettingsSingletonID = 1

// Settings stores global application settings in a singleton row (ID 1).
type Settings struct {
	bun.BaseModel `bun:"table:settings,alias:s"`

	ID                       int       `bun:"id,pk"`
	AllowSignupWithoutInvite bool      `bun:"allow_signup_without_invite,notnull,default:false"`
	FirstInitCompleted       bool      `bun:"first_init_completed,notnull,default:false"`
	CreatedAt                time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt                time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
