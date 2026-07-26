package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

// SettingsRepository persists global application settings in one row.
type SettingsRepository struct {
	db bun.IDB
}

func NewSettingsRepository(db bun.IDB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get returns the global settings row.
func (r *SettingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	settings := new(models.Settings)
	err := r.db.NewSelect().Model(settings).Where("id = ?", models.SettingsSingletonID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// SetAllowSignupWithoutInvite atomically creates or updates the signup policy.
func (r *SettingsRepository) SetAllowSignupWithoutInvite(ctx context.Context, allowed bool) (*models.Settings, error) {
	settings := &models.Settings{
		ID:                       models.SettingsSingletonID,
		AllowSignupWithoutInvite: allowed,
		UpdatedAt:                time.Now(),
	}

	_, err := r.db.NewInsert().
		Model(settings).
		Value("allow_signup_without_invite", "?", allowed).
		On("CONFLICT (id) DO UPDATE").
		Set("allow_signup_without_invite = ?", allowed).
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// SetFirstInitCompleted marks the initial admin setup as complete.
func (r *SettingsRepository) SetFirstInitCompleted(ctx context.Context, completed bool) (*models.Settings, error) {
	settings := &models.Settings{
		ID:                 models.SettingsSingletonID,
		FirstInitCompleted: completed,
		UpdatedAt:          time.Now(),
	}

	_, err := r.db.NewInsert().
		Model(settings).
		Value("first_init_completed", "?", completed).
		On("CONFLICT (id) DO UPDATE").
		Set("first_init_completed = ?", completed).
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}
