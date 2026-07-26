package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key violation")
)

// Model is the constraint for Bun model types used with BaseRepository.
// Any pointer type embedding bun.BaseModel satisfies this.
type Model[T any] interface {
	*T
}

// BaseRepository provides generic CRUD operations for any Bun model.
type BaseRepository[T any, P Model[T]] struct {
	db bun.IDB
}

func NewBaseRepository[T any, P Model[T]](db bun.IDB) *BaseRepository[T, P] {
	return &BaseRepository[T, P]{db: db}
}

func (r *BaseRepository[T, P]) Create(ctx context.Context, model P) error {
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

func (r *BaseRepository[T, P]) FindByID(ctx context.Context, id uuid.UUID) (P, error) {
	model := new(T)
	err := r.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model, nil
}

func (r *BaseRepository[T, P]) FindAll(ctx context.Context) ([]T, error) {
	var models []T
	err := r.db.NewSelect().Model(&models).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (r *BaseRepository[T, P]) Update(ctx context.Context, model P) error {
	_, err := r.db.NewUpdate().Model(model).WherePK().Exec(ctx)
	return err
}

func (r *BaseRepository[T, P]) Delete(ctx context.Context, id uuid.UUID) error {
	var zero T
	_, err := r.db.NewDelete().Model(&zero).Where("id = ?", id).Exec(ctx)
	return err
}
