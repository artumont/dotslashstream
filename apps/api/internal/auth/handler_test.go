package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artumont/dotslashstream/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSignupPolicy struct {
	allowed bool
	err     error
}

func (p mockSignupPolicy) AllowSignupWithoutInvite(context.Context) (bool, error) {
	return p.allowed, p.err
}

type mockInitializer struct {
	initialized bool
	err         error
}

func (m mockInitializer) IsInitialized(context.Context) (bool, error) {
	return m.initialized, m.err
}

func (m mockInitializer) MarkInitialized(context.Context) error {
	return m.err
}

func newAuthHandler(t *testing.T, policy mockSignupPolicy) *httptest.Server {
	return newAuthHandlerWithInit(t, policy, mockInitializer{})
}

func newAuthHandlerWithInit(t *testing.T, policy mockSignupPolicy, init mockInitializer) *httptest.Server {
	t.Helper()

	svc, _, _ := newTestService()
	handler := auth.NewHandler(svc, policy, init)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func TestHandlerRegisterRejectsInviteLessSignupWhenDisabled(t *testing.T) {
	server := newAuthHandler(t, mockSignupPolicy{allowed: false})
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register", "application/json", strings.NewReader(`{
		"username":"inviteonlyuser",
		"email":"inviteonly@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusForbidden, response.StatusCode)
}

func TestHandlerRegisterAllowsInviteLessSignupWhenEnabled(t *testing.T) {
	server := newAuthHandler(t, mockSignupPolicy{allowed: true})
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register", "application/json", strings.NewReader(`{
		"username":"opensignupuser",
		"email":"opensignup@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)
}

func TestHandlerRegisterFailsClosedWhenPolicyLookupFails(t *testing.T) {
	server := newAuthHandler(t, mockSignupPolicy{err: errors.New("database unavailable")})
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register", "application/json", strings.NewReader(`{
		"username":"policyerroruser",
		"email":"policyerror@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
}

// ── registerAdmin tests ────────────────────────────────────────────────────

func TestHandlerRegisterAdminCreatesFirstAdmin(t *testing.T) {
	handler := mockInitializer{initialized: false}
	server := newAuthHandlerWithInit(t, mockSignupPolicy{allowed: false}, handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register/admin", "application/json", strings.NewReader(`{
		"username":"firstadmin",
		"email":"admin@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	assert.NotEmpty(t, body["access_token"])
	assert.NotEmpty(t, body["refresh_token"])
}

func TestHandlerRegisterAdminReturns404WhenAlreadyInitialized(t *testing.T) {
	handler := mockInitializer{initialized: true}
	server := newAuthHandlerWithInit(t, mockSignupPolicy{allowed: true}, handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register/admin", "application/json", strings.NewReader(`{
		"username":"secondadmin",
		"email":"admin2@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestHandlerRegisterAdminReturns404WhenNilInitializer(t *testing.T) {
	svc, _, _ := newTestService()
	handler := auth.NewHandler(svc, mockSignupPolicy{}, nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register/admin", "application/json", strings.NewReader(`{
		"username":"admin",
		"email":"admin@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestHandlerRegisterAdminValidatesInput(t *testing.T) {
	handler := mockInitializer{initialized: false}
	server := newAuthHandlerWithInit(t, mockSignupPolicy{allowed: false}, handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register/admin", "application/json", strings.NewReader(`{
		"username":"",
		"email":"",
		"password":""
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestHandlerRegisterAdminRejectsSpecialCharsInUsername(t *testing.T) {
	handler := mockInitializer{initialized: false}
	server := newAuthHandlerWithInit(t, mockSignupPolicy{allowed: false}, handler)
	defer server.Close()

	response, err := http.Post(server.URL+"/auth/register/admin", "application/json", strings.NewReader(`{
		"username":"bad user!",
		"email":"admin@example.com",
		"password":"password123"
	}`))
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}
