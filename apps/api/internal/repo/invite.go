package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

// InviteRepository provides invite-specific operations.
type InviteRepository struct {
	*BaseRepository[models.Invite, *models.Invite]
}

func NewInviteRepository(db bun.IDB) *InviteRepository {
	return &InviteRepository{
		BaseRepository: NewBaseRepository[models.Invite, *models.Invite](db),
	}
}

// FindByTokenHash looks up an invite by its token hash.
func (r *InviteRepository) FindByTokenHash(ctx context.Context, tokenHash []byte) (*models.Invite, error) {
	invite := new(models.Invite)
	err := r.db.NewSelect().Model(invite).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return invite, nil
}

// IncrementUse atomically increments the use counter and returns the updated invite.
func (r *InviteRepository) IncrementUse(ctx context.Context, id uuid.UUID) (*models.Invite, error) {
	invite := new(models.Invite)
	_, err := r.db.NewUpdate().
		Model(invite).
		Set("uses = uses + 1").
		Where("id = ?", id).
		Where("uses < max_uses").
		Returning("*").
		Exec(ctx, invite)
	if err != nil {
		return nil, err
	}
	if invite.ID == uuid.Nil {
		return nil, ErrInviteExhausted
	}
	return invite, nil
}

var ErrInviteExhausted = errors.New("invite has no remaining uses")
