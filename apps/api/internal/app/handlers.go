package app

import (
	"github.com/artumont/dotslashstream/internal/auth"
	"github.com/artumont/dotslashstream/internal/repo"
	"github.com/artumont/dotslashstream/internal/settings"
)

type Handlers struct {
	auth     *auth.Handler
	settings *settings.Handler
}

// HandlerInit initializes all application handlers and their dependencies.
// It wires up authentication services including JWT, user repository,
// and invite repository, then registers the auth handler.
func (app *App) HandlerInit() error {

	// ── Handler Service Init ─────────────────────────────────────────────────────

	jwtSvc := auth.NewJWTService(app.Config.HmacSecret)
	userRepo := repo.NewUserRepository(app.Postgres.DB())
	inviteRepo := repo.NewInviteRepository(app.Postgres.DB())
	settingsRepo := repo.NewSettingsRepository(app.Postgres.DB())

	settingsService := settings.NewService(settingsRepo)
	authService := auth.NewService(jwtSvc, userRepo, inviteRepo)

	// ── Handler Init ─────────────────────────────────────────────────────────────

	app.handlers.auth = auth.NewHandler(authService, settingsService, settingsService)
	app.handlers.settings = settings.NewHandler(settingsService, authService)

	return nil
}
