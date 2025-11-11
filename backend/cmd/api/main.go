// Package main is the entry point for the pack optimizer API server.
// It initializes the application, sets up HTTP server with CORS, and handles graceful shutdown.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	httpad "github.com/temo/pack-optimizer/backend/internal/adapters/http"
	"github.com/temo/pack-optimizer/backend/internal/platform"
)

// main initializes and starts the HTTP server.
// It performs the following steps:
// 1. Configure structured logging with slog
// 2. Load configuration from environment variables
// 3. Create HTTP router with CORS middleware
// 4. Bootstrap application (connect to DB, Redis, wire dependencies)
// 5. Mount API routes
// 6. Start HTTP server in a goroutine
// 7. Wait for shutdown signal and perform graceful shutdown
func main() {
	// Configure structured logging with slog
	// Use JSON handler for production, text handler for development
	var logger *slog.Logger
	if os.Getenv("ENVIRONMENT") == "production" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	slog.SetDefault(logger)

	// Load configuration from environment variables
	cfg := platform.LoadConfig()

	// Create HTTP router
	r := chi.NewRouter()
	
	// Setup security middleware (rate limiting, DDoS protection, security headers)
	// This is part of the HTTP transport layer, so it's configured here
	httpad.SetupSecurityMiddleware(r, httpad.SecurityConfig{
		RateLimitEnabled:      cfg.RateLimitEnabled,
		RateLimitRPM:          cfg.RateLimitRPM,
		RateLimitBurst:        cfg.RateLimitBurst,
		DDoSProtectionEnabled: cfg.DDoSProtectionEnabled,
		MaxRequestSize:        cfg.MaxRequestSize,
		MaxHeaderSize:         cfg.MaxHeaderSize,
	})
	
	// CORS middleware - allow frontend access
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
	app, cleanup := platform.Bootstrap(cfg, logger)
	defer cleanup(context.Background()) // Ensure cleanup on exit

	// Create error handler for structured error responses
	errorHandler := httpad.NewErrorHandler(
		logger,
		cfg.Environment == "development",
	)

	// Mount all API routes under /api/v1 with error handling
	platform.MountRoutes(r, app, errorHandler)

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
		logger.Info("HTTP server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server crashed", "error", err)
			os.Exit(1)
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
		logger.Error("server shutdown error", "error", err)
	}
}
