package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/platform"
	minioDriver "github.com/artumont/dotslashstream/internal/platform/minio"
	pgDriver "github.com/artumont/dotslashstream/internal/platform/postgres"
	redisDriver "github.com/artumont/dotslashstream/internal/platform/redis"
)

type App struct {
	InitTime time.Time

	Config   *Config
	Redis    platform.QueueClient
	Postgres platform.DatabaseClient
	MinIO    platform.BucketClient

	server *http.Server
	Router *http.ServeMux
}

func NewApp(config *Config) *App {
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
	asynqManager := redisDriver.New(app.Config.RedisAddr)
	pgManager, err := pgDriver.New(app.Config.DatabaseDSN)
	if err != nil {
		log.Fatalf("Postgres initialization failed: %v", err)
	}
	minioManager, err := minioDriver.New(
		app.Config.BucketAddr,
		app.Config.BucketKeyID,
		app.Config.BucketAccessKey,
		app.Config.UseSSL,
	)
	if err != nil {
		log.Fatalf("Bucket initialization failed: %v", err)
	}

	app.Redis = asynqManager
	app.Postgres = pgManager
	app.MinIO = minioManager

	/*
		NOTE: Register all handler related stuff AFTER initiating the application
		services to avoid passing nil interfaces / outdated interfaces
	*/

	if err := app.RegisterAll(); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

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

	if err := app.Redis.Close(); err != nil {
		log.Printf("Redis failed to shutdown gracefully: %v", err)
	}
	if err := app.Postgres.Close(); err != nil {
		log.Printf("Postgres failed to shutdown gracefully: %v", err)
	}
	// No need to shutdown MinIO as it wraps the default http.Client

	log.Println("Server shutdown successful")
}
