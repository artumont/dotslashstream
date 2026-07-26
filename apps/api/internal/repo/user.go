package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

// UserRepository extends BaseRepository with user-specific lookups.
type UserRepository struct {
	*BaseRepository[models.User, *models.User]
}

func NewUserRepository(db bun.IDB) *UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository[models.User, *models.User](db),
	}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	user := new(models.User)
	err := r.db.NewSelect().Model(user).Where("username = ?", username).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := r.db.NewSelect().Model(user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}
