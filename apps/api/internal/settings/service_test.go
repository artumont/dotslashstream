package settings_test

import (
	"context"
	"errors"
	"testing"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
	"github.com/artumont/dotslashstream/internal/repo"
	"github.com/artumont/dotslashstream/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	settings *models.Settings
	getErr   error
	setErr   error
}

func (m *mockRepository) Get(_ context.Context) (*models.Settings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.settings == nil {
		return nil, repo.ErrNotFound
	}
	return m.settings, nil
}

func (m *mockRepository) SetAllowSignupWithoutInvite(_ context.Context, allowed bool) (*models.Settings, error) {
	if m.setErr != nil {
		return nil, m.setErr
	}
	m.settings = &models.Settings{ID: 1, AllowSignupWithoutInvite: allowed}
	return m.settings, nil
}

func (m *mockRepository) SetFirstInitCompleted(_ context.Context, completed bool) (*models.Settings, error) {
	if m.setErr != nil {
		return nil, m.setErr
	}
	if m.settings == nil {
		m.settings = &models.Settings{ID: 1}
	}
	m.settings.FirstInitCompleted = completed
	return m.settings, nil
}

func TestServiceGetUsesDefaultBeforeFirstUpdate(t *testing.T) {
	svc := settings.NewService(&mockRepository{})

	values, err := svc.Get(context.Background())
	require.NoError(t, err)
	assert.True(t, values.AllowSignupWithoutInvite)
}

func TestServiceUpdatePersistsSignupPolicy(t *testing.T) {
	repo := &mockRepository{}
	svc := settings.NewService(repo)
	allowed := false

	values, err := svc.Update(context.Background(), settings.UpdateRequest{
		AllowSignupWithoutInvite: &allowed,
	})
	require.NoError(t, err)
	assert.False(t, values.AllowSignupWithoutInvite)

	allowSignup, err := svc.AllowSignupWithoutInvite(context.Background())
	require.NoError(t, err)
	assert.False(t, allowSignup)
}

func TestServiceUpdateRequiresASetting(t *testing.T) {
	svc := settings.NewService(&mockRepository{})

	_, err := svc.Update(context.Background(), settings.UpdateRequest{})
	assert.ErrorIs(t, err, settings.ErrNoSettingsUpdated)
}

func TestServiceGetPropagatesRepositoryFailure(t *testing.T) {
	repoErr := errors.New("database unavailable")
	svc := settings.NewService(&mockRepository{getErr: repoErr})

	_, err := svc.Get(context.Background())
	assert.ErrorIs(t, err, repoErr)
}
