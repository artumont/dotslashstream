package app

import (
	"log"
	"net/http"
)

type RouteRegistrar interface {
	// Register adds all routes handled by this registrar to the given ServeMux.
	RegisterRoutes(mux *http.ServeMux)
}

// RegisterAll iterates over every previously registered RouteRegistrar and
// wires its routes into the provided http.ServeMux.
func (app *App) RegisterAll() error {
	log.Println("Registering routes...")

	/*
		NOTE: Register all handlers here, all handlers should have the RouteRegistrar
		interface, make sure to include logging so error diagnosis goes smoothly
	*/

	// ── Auth Routes ──────────────────────────────────────────────────────────────

	if app.handlers.auth != nil {
		app.handlers.auth.RegisterRoutes(app.router)
	}
	log.Println("  Registered auth routes")

	// ── Settings Routes ──────────────────────────────────────────────────────────

	if app.handlers.settings != nil {
		app.handlers.settings.RegisterRoutes(app.router)
	}
	log.Println("  Registered settings routes")

	return nil
}
