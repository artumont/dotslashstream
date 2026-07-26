package models_test

import (
	"context"
	"testing"

	"github.com/artumont/dotslashstream/internal/platform/postgres/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserBeforeAppendModelGeneratesUUIDv7(t *testing.T) {
	user := new(models.User)

	require.NoError(t, user.BeforeAppendModel(context.Background(), nil))
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, uuid.Version(7), user.ID.Version())
}

func TestInviteBeforeAppendModelGeneratesUUIDv7(t *testing.T) {
	invite := new(models.Invite)

	require.NoError(t, invite.BeforeAppendModel(context.Background(), nil))
	require.NotEqual(t, uuid.Nil, invite.ID)
	require.Equal(t, uuid.Version(7), invite.ID.Version())
}

func TestBeforeAppendModelPreservesExplicitIDs(t *testing.T) {
	existing := uuid.MustParse("01956b5d-5f40-7000-8000-000000000001")
	user := &models.User{ID: existing}
	invite := &models.Invite{ID: existing}

	require.NoError(t, user.BeforeAppendModel(context.Background(), nil))
	require.NoError(t, invite.BeforeAppendModel(context.Background(), nil))
	require.Equal(t, existing, user.ID)
	require.Equal(t, existing, invite.ID)
}
