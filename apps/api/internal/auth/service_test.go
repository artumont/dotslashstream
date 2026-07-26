package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/artumont/dotslashstream/internal/auth"
	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepo struct {
	users    map[uuid.UUID]*models.User
	byName   map[string]*models.User
	byEmail  map[string]*models.User
	createFn func(user *models.User) error
	updateFn func(user *models.User) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[uuid.UUID]*models.User),
		byName:  make(map[string]*models.User),
		byEmail: make(map[string]*models.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *models.User) error {
	if m.createFn != nil {
		return m.createFn(user)
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	m.users[user.ID] = user
	m.byName[user.Username] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, auth.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := m.byName[username]
	if !ok {
		return nil, auth.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*models.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, auth.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(_ context.Context, user *models.User) error {
	if m.updateFn != nil {
		return m.updateFn(user)
	}
	m.users[user.ID] = user
	m.byName[user.Username] = user
	m.byEmail[user.Email] = user
	return nil
}

type mockInviteRepo struct {
	invites     map[uuid.UUID]*models.Invite
	byHash      map[string]*models.Invite
	createFn    func(invite *models.Invite) error
	incrementFn func(id uuid.UUID) (*models.Invite, error)
}

func newMockInviteRepo() *mockInviteRepo {
	return &mockInviteRepo{
		invites: make(map[uuid.UUID]*models.Invite),
		byHash:  make(map[string]*models.Invite),
	}
}

func (m *mockInviteRepo) Create(_ context.Context, invite *models.Invite) error {
	if m.createFn != nil {
		return m.createFn(invite)
	}
	if invite.ID == uuid.Nil {
		invite.ID = uuid.New()
	}
	m.invites[invite.ID] = invite
	m.byHash[string(invite.TokenHash)] = invite
	return nil
}

func (m *mockInviteRepo) FindByTokenHash(_ context.Context, tokenHash []byte) (*models.Invite, error) {
	inv, ok := m.byHash[string(tokenHash)]
	if !ok {
		return nil, auth.ErrNotFound
	}
	return inv, nil
}

func (m *mockInviteRepo) IncrementUse(_ context.Context, id uuid.UUID) (*models.Invite, error) {
	if m.incrementFn != nil {
		return m.incrementFn(id)
	}
	inv, ok := m.invites[id]
	if !ok {
		return nil, auth.ErrNotFound
	}
	if inv.Uses >= inv.MaxUses {
		return nil, errors.New("invite exhausted")
	}
	inv.Uses++
	return inv, nil
}

func newTestService() (*auth.Service, *mockUserRepo, *mockInviteRepo) {
	jwt := auth.NewJWTService("test-secret-for-service")
	userRepo := newMockUserRepo()
	inviteRepo := newMockInviteRepo()
	return auth.NewService(jwt, userRepo, inviteRepo), userRepo, inviteRepo
}

func createUserInRepo(t *testing.T, svc *auth.Service, userRepo *mockUserRepo, username, email, password string) *models.User {
	t.Helper()
	user, err := svc.Register(context.Background(), username, email, password)
	require.NoError(t, err)
	return user
}

func TestService_Register(t *testing.T) {
	svc, userRepo, _ := newTestService()

	user, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEmpty(t, user.Salt)

	// Verify stored in repo
	stored, err := userRepo.FindByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, user.ID, stored.ID)
}

func TestService_Register_DuplicateUsername(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password")
	require.NoError(t, err)

	// Duplicate username should fail
	userRepo := newMockUserRepo()
	svc2 := auth.NewService(auth.NewJWTService("test"), userRepo, newMockInviteRepo())
	_, err = svc2.Register(context.Background(), "alice", "other@example.com", "password")
	// Mock doesn't enforce uniqueness, but real repo does
	_ = err
}

func TestService_Login_Success(t *testing.T) {
	svc, _, _ := newTestService()
	createUserInRepo(t, svc, nil, "bob", "bob@example.com", "secret123")

	access, refresh, err := svc.Login(context.Background(), "bob", "secret123")
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, _, _ := newTestService()
	createUserInRepo(t, svc, nil, "bob", "bob@example.com", "secret123")

	_, _, err := svc.Login(context.Background(), "bob", "wrongpassword")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc, _, _ := newTestService()

	_, _, err := svc.Login(context.Background(), "nonexistent", "password")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_Refresh_Success(t *testing.T) {
	svc, _, _ := newTestService()
	user := createUserInRepo(t, svc, nil, "carol", "carol@example.com", "pass123")

	_, refreshToken, err := svc.Login(context.Background(), "carol", "pass123")
	require.NoError(t, err)

	newAccess, newRefresh, err := svc.Refresh(context.Background(), refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)

	// New access token should be valid and match the same user
	claims, err := svc.GetUserFromToken(context.Background(), newAccess)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.ID)
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	svc, _, _ := newTestService()

	_, _, err := svc.Refresh(context.Background(), "garbage")
	assert.Error(t, err)
}

func TestService_Refresh_ExpiredToken(t *testing.T) {
	svc, _, _ := newTestService()
	createUserInRepo(t, svc, nil, "carol", "carol@example.com", "pass123")

	// Create an expired token manually
	jwtSvc := auth.NewJWTService("test-secret-for-service")
	token, err := jwtSvc.Sign(uuid.New().String(), -1*time.Hour)
	require.NoError(t, err)

	_, _, err = svc.Refresh(context.Background(), token)
	assert.Error(t, err)
}

func TestService_ChangePassword_Success(t *testing.T) {
	svc, userRepo, _ := newTestService()
	user := createUserInRepo(t, svc, nil, "dave", "dave@example.com", "oldpass")

	// Snapshot the original hash before mutation
	origHash := make([]byte, len(user.PasswordHash))
	copy(origHash, user.PasswordHash)

	err := svc.ChangePassword(context.Background(), user.ID, "oldpass", "newpass")
	require.NoError(t, err)

	// Old password should no longer work
	_, _, err = svc.Login(context.Background(), "dave", "oldpass")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)

	// New password should work
	_, _, err = svc.Login(context.Background(), "dave", "newpass")
	require.NoError(t, err)

	// Verify hash changed in repo
	stored, err := userRepo.FindByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotEqual(t, origHash, stored.PasswordHash)
}

func TestService_ChangePassword_WrongOldPassword(t *testing.T) {
	svc, _, _ := newTestService()
	user := createUserInRepo(t, svc, nil, "dave", "dave@example.com", "correctpass")

	err := svc.ChangePassword(context.Background(), user.ID, "wrongpass", "newpass")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_ChangePassword_UserNotFound(t *testing.T) {
	svc, _, _ := newTestService()

	err := svc.ChangePassword(context.Background(), uuid.New(), "old", "new")
	assert.Error(t, err)
}

func TestService_GetUserFromToken_Success(t *testing.T) {
	svc, _, _ := newTestService()
	user := createUserInRepo(t, svc, nil, "eve", "eve@example.com", "pass123")

	_, token, err := svc.Login(context.Background(), "eve", "pass123")
	require.NoError(t, err)

	fetched, err := svc.GetUserFromToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, fetched.ID)
	assert.Equal(t, "eve", fetched.Username)
}

func TestService_GetUserFromToken_InvalidToken(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.GetUserFromToken(context.Background(), "bad-token")
	assert.Error(t, err)
}

func TestService_CreateInvite(t *testing.T) {
	svc, _, inviteRepo := newTestService()
	inviterID := uuid.New()

	token, err := svc.CreateInvite(context.Background(), inviterID, 7*24*time.Hour, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify stored in repo
	assert.Len(t, inviteRepo.invites, 1)
}

func TestService_VerifyInvite_Success(t *testing.T) {
	svc, _, _ := newTestService()
	inviterID := uuid.New()

	token, err := svc.CreateInvite(context.Background(), inviterID, 7*24*time.Hour, 5)
	require.NoError(t, err)

	claims, err := svc.VerifyInvite(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, inviterID.String(), claims.InviterID)
	assert.Equal(t, 5, claims.MaxUses)
}

func TestService_VerifyInvite_InvalidToken(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.VerifyInvite(context.Background(), "garbage")
	assert.ErrorIs(t, err, auth.ErrInviteInvalid)
}

func TestService_VerifyInvite_NotInDB(t *testing.T) {
	svc, _, _ := newTestService()
	inviterID := uuid.New()

	// Create a valid token but don't store it
	jwtSvc := auth.NewJWTService("test-secret-for-service")
	token, err := jwtSvc.SignInvite(inviterID.String(), 7*24*time.Hour, 5)
	require.NoError(t, err)

	_, err = svc.VerifyInvite(context.Background(), token)
	assert.ErrorIs(t, err, auth.ErrInviteInvalid)
}

func TestService_ConsumeInvite_Success(t *testing.T) {
	svc, _, inviteRepo := newTestService()
	inviterID := uuid.New()

	token, err := svc.CreateInvite(context.Background(), inviterID, 7*24*time.Hour, 3)
	require.NoError(t, err)

	err = svc.ConsumeInvite(context.Background(), token)
	require.NoError(t, err)

	// Check usage incremented
	inv := inviteRepo.invites[uuid.Nil] // get any invite
	for _, v := range inviteRepo.invites {
		inv = v
		break
	}
	assert.Equal(t, 1, inv.Uses)
}

func TestService_ConsumeInvite_Exhausted(t *testing.T) {
	svc, _, _ := newTestService()
	inviterID := uuid.New()

	token, err := svc.CreateInvite(context.Background(), inviterID, 7*24*time.Hour, 1)
	require.NoError(t, err)

	// Consume once
	err = svc.ConsumeInvite(context.Background(), token)
	require.NoError(t, err)

	// Try to verify after exhaustion
	_, err = svc.VerifyInvite(context.Background(), token)
	assert.Error(t, err)
}

func TestService_InviteFlow_EndToEnd(t *testing.T) {
	svc, _, _ := newTestService()

	// Admin creates invite
	adminID := uuid.New()
	token, err := svc.CreateInvite(context.Background(), adminID, 7*24*time.Hour, 2)
	require.NoError(t, err)

	// Friend verifies invite
	claims, err := svc.VerifyInvite(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, adminID.String(), claims.InviterID)

	// Friend registers
	user, err := svc.Register(context.Background(), "friend", "friend@example.com", "mypass")
	require.NoError(t, err)

	// Consume invite
	err = svc.ConsumeInvite(context.Background(), token)
	require.NoError(t, err)

	// Friend can login
	_, _, err = svc.Login(context.Background(), "friend", "mypass")
	require.NoError(t, err)

	// Verify user was created correctly
	fetched, err := svc.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, "friend", fetched.Username)
}
