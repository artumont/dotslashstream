package settings_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/artumont/dotslashstream/internal/auth"
	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
	"github.com/artumont/dotslashstream/internal/settings"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authUserRepository struct {
	user *models.User
}

func (r *authUserRepository) Create(context.Context, *models.User) error {
	return errors.New("not implemented")
}
func (r *authUserRepository) FindByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, auth.ErrNotFound
}
func (r *authUserRepository) FindByUsername(context.Context, string) (*models.User, error) {
	return nil, auth.ErrNotFound
}
func (r *authUserRepository) FindByEmail(context.Context, string) (*models.User, error) {
	return nil, auth.ErrNotFound
}
func (r *authUserRepository) Update(context.Context, *models.User) error {
	return errors.New("not implemented")
}

type authInviteRepository struct{}

func (authInviteRepository) Create(context.Context, *models.Invite) error {
	return errors.New("not implemented")
}
func (authInviteRepository) FindByTokenHash(context.Context, []byte) (*models.Invite, error) {
	return nil, auth.ErrNotFound
}
func (authInviteRepository) IncrementUse(context.Context, uuid.UUID) (*models.Invite, error) {
	return nil, auth.ErrNotFound
}

func newSettingsServer(t *testing.T, isAdmin bool) (*httptest.Server, string) {
	t.Helper()

	user := &models.User{ID: uuid.New(), IsAdmin: isAdmin}
	jwt := auth.NewJWTService("settings-handler-test-secret")
	authSvc := auth.NewService(jwt, &authUserRepository{user: user}, authInviteRepository{})
	settingsSvc := settings.NewService(&mockRepository{})
	handler := settings.NewHandler(settingsSvc, authSvc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	token, err := jwt.Sign(user.ID.String(), time.Minute)
	require.NoError(t, err)
	return httptest.NewServer(mux), token
}

func TestHandlerAllowsAdminToReadAndUpdateSettings(t *testing.T) {
	server, token := newSettingsServer(t, true)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/settings", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var current settings.Values
	require.NoError(t, json.NewDecoder(response.Body).Decode(&current))
	assert.True(t, current.AllowSignupWithoutInvite)

	request, err = http.NewRequest(http.MethodPatch, server.URL+"/settings", strings.NewReader(`{"allow_signup_without_invite":false}`))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	var updated settings.Values
	require.NoError(t, json.NewDecoder(response.Body).Decode(&updated))
	assert.False(t, updated.AllowSignupWithoutInvite)
}

func TestHandlerRejectsUnauthenticatedAndNonAdminRequests(t *testing.T) {
	server, token := newSettingsServer(t, false)
	defer server.Close()

	response, err := http.Get(server.URL + "/settings")
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)

	request, err := http.NewRequest(http.MethodGet, server.URL+"/settings", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusForbidden, response.StatusCode)
}
