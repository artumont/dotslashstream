package models

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Invite tracks usage state for partially-stateless invite tokens.
type Invite struct {
	bun.BaseModel `bun:"table:invites,alias:i"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	TokenHash []byte    `bun:"token_hash,notnull,unique"`
	MaxUses   int       `bun:"max_uses,notnull"`
	Uses      int       `bun:"uses,notnull,default:0"`
	CreatedBy uuid.UUID `bun:"created_by,notnull,type:uuid"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*Invite)(nil)

// BeforeAppendModel generates a UUID v7 before the model is inserted.
func (i *Invite) BeforeAppendModel(_ context.Context, _ bun.Query) error {
	if i.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		i.ID = id
	}
	return nil
}

// HashToken returns the SHA-256 hash of an invite token for storage.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// TokenHashString returns the hash as a hex string.
func TokenHashString(token string) string {
	return fmt.Sprintf("%x", HashToken(token))
}
