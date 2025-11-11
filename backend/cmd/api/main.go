// Package main is the entry point for the pack optimizer API server.
// It initializes the application, sets up HTTP server with CORS, and handles graceful shutdown.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/temo/pack-optimizer/backend/internal/platform"
)

// main initializes and starts the HTTP server.
// It performs the following steps:
// 1. Configure structured logging with zerolog
// 2. Load configuration from environment variables
// 3. Create HTTP router with CORS middleware
// 4. Bootstrap application (connect to DB, Redis, wire dependencies)
// 5. Mount API routes
// 6. Start HTTP server in a goroutine
// 7. Wait for shutdown signal and perform graceful shutdown
func main() {
	// Configure structured logging with console-friendly output
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// Load configuration from environment variables
	cfg := platform.LoadConfig()

	// Create HTTP router
	r := chi.NewRouter()
	
	// Configure CORS middleware to allow frontend access
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins (configure for production)
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Cache preflight requests for 5 minutes
	}))

	// Root endpoint - redirects to API v1
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/v1", http.StatusMovedPermanently)
	})
	
	// API root endpoint - redirects to API v1
	r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/v1", http.StatusMovedPermanently)
	})

	// Bootstrap application: connect to dependencies and wire services
	app, cleanup := platform.Bootstrap(cfg)
	defer cleanup(context.Background()) // Ensure cleanup on exit

	// Mount all API routes under /api/v1
	platform.MountRoutes(r, app)

	// Configure HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,  // Maximum time to read request
		WriteTimeout: 15 * time.Second,  // Maximum time to write response
		IdleTimeout:  60 * time.Second,  // Maximum time to wait for next request
	}

	// Start server in a goroutine to allow graceful shutdown handling
	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server crashed")
		}
	}()

	// Graceful shutdown: wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop // Block until signal received
	
	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}
}
