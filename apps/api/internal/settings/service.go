package settings

import (
	"context"
	"errors"

	"github.com/artumont/dotslashstream/internal/repo"
)

const defaultAllowSignupWithoutInvite = true

var ErrNoSettingsUpdated = errors.New("no settings supplied")

// Service owns global-settings defaults and updates.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns all current values, applying defaults before first persistence.
func (s *Service) Get(ctx context.Context) (Values, error) {
	settings, err := s.repo.Get(ctx)
	if errors.Is(err, repo.ErrNotFound) {
		return Values{
			AllowSignupWithoutInvite: defaultAllowSignupWithoutInvite,
		}, nil
	}
	if err != nil {
		return Values{}, err
	}
	return Values{
		AllowSignupWithoutInvite: settings.AllowSignupWithoutInvite,
		FirstInitCompleted:       settings.FirstInitCompleted,
	}, nil
}

// IsInitialized returns true if the initial admin has already been created.
func (s *Service) IsInitialized(ctx context.Context) (bool, error) {
	values, err := s.Get(ctx)
	if err != nil {
		return false, err
	}
	return values.FirstInitCompleted, nil
}

// MarkInitialized sets the first_init_completed flag to true.
func (s *Service) MarkInitialized(ctx context.Context) error {
	_, err := s.repo.SetFirstInitCompleted(ctx, true)
	return err
}

// AllowSignupWithoutInvite implements auth's signup-policy dependency.
func (s *Service) AllowSignupWithoutInvite(ctx context.Context) (bool, error) {
	values, err := s.Get(ctx)
	if err != nil {
		return false, err
	}
	return values.AllowSignupWithoutInvite, nil
}

// Update persists supplied settings and returns the full settings document.
func (s *Service) Update(ctx context.Context, update UpdateRequest) (Values, error) {
	if update.AllowSignupWithoutInvite == nil {
		return Values{}, ErrNoSettingsUpdated
	}

	settings, err := s.repo.SetAllowSignupWithoutInvite(ctx, *update.AllowSignupWithoutInvite)
	if err != nil {
		return Values{}, err
	}
	return Values{
		AllowSignupWithoutInvite: settings.AllowSignupWithoutInvite,
		FirstInitCompleted:       settings.FirstInitCompleted,
	}, nil
}
