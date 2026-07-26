package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User is the database-level model for the users table.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           uuid.UUID `bun:"id,pk,type:uuid"`
	Username     string    `bun:"username,notnull,unique"`
	Email        string    `bun:"email,notnull,unique"`
	PasswordHash []byte    `bun:"password_hash,notnull"`
	Salt         []byte    `bun:"salt,notnull,type:bytea"`
	CreatedAt    time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	IsAdmin      bool      `bun:"is_admin,notnull,default:false"`
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

// BeforeAppendModel generates a UUID v7 before the model is inserted.
func (u *User) BeforeAppendModel(_ context.Context, _ bun.Query) error {
	if u.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		u.ID = id
	}
	return nil
}
