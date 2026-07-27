package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/artumont/dotslashstream/internal/middleware"
	"github.com/artumont/dotslashstream/internal/platform"
	asynqDriver "github.com/artumont/dotslashstream/internal/platform/asynq"
	minioDriver "github.com/artumont/dotslashstream/internal/platform/minio"
	pgDriver "github.com/artumont/dotslashstream/internal/platform/postgres"
	redisDriver "github.com/artumont/dotslashstream/internal/platform/redis"
)

type App struct {
	InitTime time.Time

	Config   *Config
	Redis    platform.RedisClient
	Queue    platform.QueueClient
	Postgres platform.DatabaseClient
	MinIO    platform.BucketClient

	server   *http.Server
	router   *http.ServeMux
	handlers Handlers
}

func NewApp(config *Config) *App {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
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

		server:   server,
		router:   router,
		handlers: Handlers{},
	}
}

// Start initializes the application services and starts
// serving without blocking the main execution loop
func (app *App) Start() <-chan error {
	log.Println("Initializing server dependencies and routes...")
	redisManager := redisDriver.New(app.Config.RedisAddr)
	asynqManager := asynqDriver.New(app.Config.RedisAddr)
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

	app.Redis = redisManager
	app.Queue = asynqManager
	app.Postgres = pgManager
	app.MinIO = minioManager

	/*
		NOTE: Register all handler related stuff AFTER initiating the application
		services to avoid passing nil interfaces / outdated interfaces
	*/

	if err := app.HandlerInit(); err != nil {
		log.Fatalf("Failed to init handlers: %v", err)
	}
	if err := app.RegisterAllHandlers(); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	// Wrap router with global middleware (logger, rate limit).
	app.server.Handler = middleware.Chain(app.router, app.Redis)

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
