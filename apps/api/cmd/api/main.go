package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/artumont/dotslashstream/internal/app"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load() // load .env if present, ignore error if missing

	environment, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("Environment loading failed: %v", err)
	}
	app := app.NewApp(environment)
	serverErrors := app.Start()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Critical server error: %v", err)
	case sig := <-shutdownSignals:
		log.Printf("Received shutdown signal (%v). Starting graceful termination...", sig)
		app.Shutdown()
	}
}
