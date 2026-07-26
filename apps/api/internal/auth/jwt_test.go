package auth_test

import (
	"testing"
	"time"

	"github.com/artumont/dotslashstream/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJWTService() *auth.JWTService {
	return auth.NewJWTService("test-secret-key-for-testing")
}

func TestJWT_SignVerify(t *testing.T) {
	svc := newJWTService()
	userID := uuid.New().String()

	token, err := svc.Sign(userID, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "dotslashstream", claims.Issuer)
	assert.NotEmpty(t, claims.ID)
}

func TestJWT_VerifyExpired(t *testing.T) {
	svc := newJWTService()
	userID := uuid.New().String()

	// Sign with negative TTL = already expired
	token, err := svc.Sign(userID, -1*time.Hour)
	require.NoError(t, err)

	_, err = svc.Verify(token)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestJWT_VerifyInvalid(t *testing.T) {
	svc := newJWTService()

	_, err := svc.Verify("not-a-real-token")
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}

func TestJWT_VerifyWrongSecret(t *testing.T) {
	svc1 := auth.NewJWTService("secret-one")
	svc2 := auth.NewJWTService("secret-two")
	userID := uuid.New().String()

	token, err := svc1.Sign(userID, 15*time.Minute)
	require.NoError(t, err)

	_, err = svc2.Verify(token)
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}

func TestJWT_SignVerifyInvite(t *testing.T) {
	svc := newJWTService()
	inviterID := uuid.New().String()
	maxUses := 5

	token, err := svc.SignInvite(inviterID, 7*24*time.Hour, maxUses)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.VerifyInvite(token)
	require.NoError(t, err)
	assert.Equal(t, inviterID, claims.InviterID)
	assert.Equal(t, maxUses, claims.MaxUses)
	assert.Equal(t, "dotslashstream-invite", claims.Issuer)
	assert.NotEmpty(t, claims.ID)
}

func TestJWT_SignProducesUniqueTokens(t *testing.T) {
	svc := newJWTService()
	userID := uuid.New().String()

	first, err := svc.Sign(userID, 15*time.Minute)
	require.NoError(t, err)
	second, err := svc.Sign(userID, 15*time.Minute)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestJWT_VerifyInviteExpired(t *testing.T) {
	svc := newJWTService()
	inviterID := uuid.New().String()

	token, err := svc.SignInvite(inviterID, -1*time.Hour, 5)
	require.NoError(t, err)

	_, err = svc.VerifyInvite(token)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestJWT_VerifyInviteInvalid(t *testing.T) {
	svc := newJWTService()

	_, err := svc.VerifyInvite("garbage-token")
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}
