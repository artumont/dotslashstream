package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	models "github.com/artumont/dotslashstream/internal/platform/postgres/models"
)

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 7 * 24 * time.Hour
	maxBCryptCost          = 14
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInviteExpired      = errors.New("invite has expired")
	ErrInviteExhausted    = errors.New("invite has no remaining uses")
	ErrInviteInvalid      = errors.New("invalid invite token")
	ErrNotFound           = errors.New("record not found")
)

// Service holds auth business logic.
type Service struct {
	jwt        *JWTService
	userRepo   UserRepo
	inviteRepo InviteRepo
}

func NewService(jwt *JWTService, userRepo UserRepo, inviteRepo InviteRepo) *Service {
	return &Service{
		jwt:        jwt,
		userRepo:   userRepo,
		inviteRepo: inviteRepo,
	}
}

// Register creates a new user with the given credentials.
func (s *Service) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	return s.registerWithRole(ctx, username, email, password, false)
}

// RegisterAdmin creates the initial admin user.
// Only call this during first-time setup when no admin exists.
func (s *Service) RegisterAdmin(ctx context.Context, username, email, password string) (*models.User, error) {
	return s.registerWithRole(ctx, username, email, password, true)
}

func (s *Service) registerWithRole(ctx context.Context, username, email, password string, isAdmin bool) (*models.User, error) {
	salt, hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Salt:         salt,
		IsAdmin:      isAdmin,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login verifies credentials and returns access + refresh tokens.
func (s *Service) Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := checkPassword(password, user.Salt, user.PasswordHash); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}

	accessToken, err = s.jwt.Sign(user.ID.String(), defaultAccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = s.jwt.Sign(user.ID.String(), defaultRefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// Refresh verifies a refresh token and issues a new token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error) {
	claims, err := s.jwt.Verify(refreshToken)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if _, err := s.userRepo.FindByID(ctx, userID); err != nil {
		return "", "", ErrInvalidCredentials
	}

	newAccess, err = s.jwt.Sign(userID.String(), defaultAccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	newRefresh, err = s.jwt.Sign(userID.String(), defaultRefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return newAccess, newRefresh, nil
}

// ChangePassword verifies the old password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := checkPassword(oldPassword, user.Salt, user.PasswordHash); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return err
	}

	salt, hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Salt = salt
	user.PasswordHash = hash
	return s.userRepo.Update(ctx, user)
}

// GetUserByID returns the user for the given ID.
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// GetUserFromToken verifies a JWT and returns the corresponding user.
func (s *Service) GetUserFromToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := s.jwt.Verify(token)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}

	return s.userRepo.FindByID(ctx, userID)
}

// CreateInvite generates a partially-stateless invite token.
//
// The token itself is a signed JWT containing inviter ID, max uses, and expiry.
// Usage state is tracked in the invites table via SHA-256(token).
func (s *Service) CreateInvite(ctx context.Context, inviterID uuid.UUID, ttl time.Duration, maxUses int) (token string, err error) {
	token, err = s.jwt.SignInvite(inviterID.String(), ttl, maxUses)
	if err != nil {
		return "", err
	}

	invite := &models.Invite{
		TokenHash: models.HashToken(token),
		MaxUses:   maxUses,
		Uses:      0,
		CreatedBy: inviterID,
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		return "", err
	}

	return token, nil
}

// VerifyInvite checks an invite token's validity and remaining uses.
func (s *Service) VerifyInvite(ctx context.Context, token string) (*InviteClaims, error) {
	claims, err := s.jwt.VerifyInvite(token)
	if err != nil {
		return nil, ErrInviteInvalid
	}

	tokenHash := models.HashToken(token)
	invite, err := s.inviteRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInviteInvalid
	}

	if time.Now().After(time.Unix(claims.ExpiresAt.Unix(), 0)) {
		return nil, ErrInviteExpired
	}

	if invite.Uses >= invite.MaxUses {
		return nil, ErrInviteExhausted
	}

	return claims, nil
}

// ConsumeInvite atomically increments the usage counter for an invite.
// Call after successful registration to prevent race conditions.
func (s *Service) ConsumeInvite(ctx context.Context, token string) error {
	tokenHash := models.HashToken(token)
	invite, err := s.inviteRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	_, err = s.inviteRepo.IncrementUse(ctx, invite.ID)
	return err
}
