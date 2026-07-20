package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/bucket"
	"github.com/artumont/dotslashstream/internal/database/postgres"
	"github.com/artumont/dotslashstream/internal/database/redis"
)

type App struct {
	InitTime time.Time

	Config   *Environment
	Redis    redis.QueueClient
	Postgres postgres.PostgresManager
	MinIO    bucket.BucketManager

	server *http.Server
	Router *http.ServeMux
}

func NewApp(config *Environment) *App {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return &App{
		InitTime: time.Now(),
		Config:   config,
		Redis:    nil,
		Postgres: nil,
		MinIO:    nil,

		server: server,
		Router: router,
	}
}

// Start initializes the application services and starts
// serving without blocking the main execution loop
func (app *App) Start() <-chan error {
	log.Println("Initializing server dependencies and routes...")

	// NOTE: Initialize services and routes here

	asynqManager := redis.NewAsyncqClient(app.Config.RedisAddr)
	pgManager, err := postgres.NewBunPostgresManager(app.Config.DatabaseDSN)
	if err != nil {
		log.Fatalf("Postgres initialization failed: %v", err)
	}
	minioManager, err := bucket.NewMinioBucketManager(app.Config.BucketAddr, app.Config.BucketKeyID, app.Config.BucketAccessKey, false)
	if err != nil {
		log.Fatalf("Bucket initialization failed: %v", err)
	}

	app.Redis = asynqManager
	app.Postgres = pgManager
	app.MinIO = minioManager

	errChannel := make(chan error, 1)

	// Run ListenAndServe in the background so it doesn't block execution
	go func() {
		log.Printf("Starting server on `%s`", app.server.Addr)
		err := app.server.ListenAndServe()

		// If it's a normal closure, don't bubble it up as an error
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChannel <- err
		}
		close(errChannel)
	}()

	return errChannel
}

// GracefulShutdown attempts to gracefully shutdown the services and server with a timeout
// If the shutdown timeouts it defaults to a normal shutdown
func (app *App) Shutdown() {
	log.Println("Initializing graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed, forcing close: %v", err)
		_ = app.server.Close() // Fallback to force-closing if it times out
	}

	/*
		NOTE: Clean up internal services and infra deps here AFTER server shutdown
		This is to avoid interrupting ongoing transactions that require the services
	*/

	log.Println("Server shutdown successful")
}
