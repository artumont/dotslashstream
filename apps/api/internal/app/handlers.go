package app

import (
	"log"

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
	log.Println("Initializing handlers...")

	// ── Handler Repo Init ────────────────────────────────────────────────────────

	userRepo := repo.NewUserRepository(app.Postgres.DB())
	inviteRepo := repo.NewInviteRepository(app.Postgres.DB())
	settingsRepo := repo.NewSettingsRepository(app.Postgres.DB())
	log.Println("  Initialized all repos")

	// ── Handler Service Init ─────────────────────────────────────────────────────

	jwtSvc := auth.NewJWTService(app.Config.HmacSecret)
	settingsService := settings.NewService(settingsRepo)
	authService := auth.NewService(jwtSvc, userRepo, inviteRepo)
	log.Println("  Initialized all handler services")

	// ── Handler Init ─────────────────────────────────────────────────────────────

	app.handlers.auth = auth.NewHandler(authService, settingsService, settingsService)
	log.Println("  Initialized auth handler")
	app.handlers.settings = settings.NewHandler(settingsService, authService)
	log.Println("  Initialized settings handler")
	return nil
}
