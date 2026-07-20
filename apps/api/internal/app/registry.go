package app

import "net/http"

type RouteRegistrar interface {
	// Register adds all routes handled by this registrar to the given ServeMux.
	RegisterRoutes(mux *http.ServeMux)
}

// RegisterAll iterates over every previously registered RouteRegistrar and
// wires its routes into the provided http.ServeMux.
func (app *App) RegisterAll() error {
	/*
		NOTE: Register all handlers here, all handlers should have the RouteRegistrar
		interface, make sure to include logging so error diagnosis goes smoothly
	*/

	return nil
}
