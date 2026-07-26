package auth

import (
	"context"

	"github.com/google/uuid"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

// UserRepo defines the user repository interface for testability.
type UserRepo interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

// InviteRepo defines the invite repository interface for testability.
type InviteRepo interface {
	Create(ctx context.Context, invite *models.Invite) error
	FindByTokenHash(ctx context.Context, tokenHash []byte) (*models.Invite, error)
	IncrementUse(ctx context.Context, id uuid.UUID) (*models.Invite, error)
}
