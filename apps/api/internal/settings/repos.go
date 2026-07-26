package settings

import (
	"context"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

// Repository defines persistence required by the settings service.
type Repository interface {
	Get(ctx context.Context) (*models.Settings, error)
	SetAllowSignupWithoutInvite(ctx context.Context, allowed bool) (*models.Settings, error)
	SetFirstInitCompleted(ctx context.Context, completed bool) (*models.Settings, error)
}
