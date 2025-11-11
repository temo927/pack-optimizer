// Package platform provides dependency injection and application bootstrapping.
// This file contains recovery mechanisms for external dependencies.
package platform

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// RetryConfig holds configuration for retry logic.
type RetryConfig struct {
	MaxAttempts       int           // Maximum number of retry attempts
	InitialDelay     time.Duration // Initial delay before first retry
	MaxDelay         time.Duration // Maximum delay between retries
	BackoffMultiplier float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic.
// Returns the result of the function or an error if all retries are exhausted.
func RetryWithBackoff(ctx context.Context, logger *slog.Logger, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			if attempt > 0 {
				logger.Info("operation succeeded after retry", "attempt", attempt+1)
			}
			return nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Wait before retry
		logger.Warn(
			"operation failed, retrying",
			"error", err.Error(),
			"attempt", attempt+1,
			"max_attempts", config.MaxAttempts,
			"delay", delay,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Exponential backoff
			delay = time.Duration(float64(delay) * config.BackoffMultiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return lastErr
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota // Normal operation
	CircuitBreakerOpen                              // Failing, reject requests
	CircuitBreakerHalfOpen                          // Testing if service recovered
)

// CircuitBreaker implements the circuit breaker pattern for external dependencies.
type CircuitBreaker struct {
	logger          *slog.Logger
	maxFailures     int
	resetTimeout    time.Duration
	state           CircuitBreakerState
	failureCount    int
	lastFailureTime time.Time
	successCount    int // For half-open state
	halfOpenRequests int // Number of requests to test in half-open state
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(logger *slog.Logger, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		logger:          logger,
		maxFailures:     maxFailures,
		resetTimeout:    resetTimeout,
		state:           CircuitBreakerClosed,
		halfOpenRequests: 3, // Test with 3 requests in half-open state
	}
}

// Execute executes a function through the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.updateState()

	switch cb.state {
	case CircuitBreakerOpen:
		return errors.New("circuit breaker is open - service unavailable")
	case CircuitBreakerHalfOpen:
		// Allow limited requests to test if service recovered
		if cb.successCount >= cb.halfOpenRequests {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.logger.Info("circuit breaker closed - service recovered")
		}
		fallthrough
	case CircuitBreakerClosed:
		err := fn()
		if err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil
	}

	return nil
}

// updateState updates the circuit breaker state based on time and failure count.
func (cb *CircuitBreaker) updateState() {
	now := time.Now()

	switch cb.state {
	case CircuitBreakerOpen:
		// Check if reset timeout has passed
		if now.Sub(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.successCount = 0
			cb.logger.Info("circuit breaker half-open - testing service recovery")
		}
	case CircuitBreakerHalfOpen:
		// Already handled in Execute
	case CircuitBreakerClosed:
		// Check if we should open the circuit
		if cb.failureCount >= cb.maxFailures {
			cb.state = CircuitBreakerOpen
			cb.lastFailureTime = now
			cb.logger.Error(
				"circuit breaker opened - too many failures",
				"failures", cb.failureCount,
			)
		}
	}
}

// recordFailure records a failure in the circuit breaker.
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == CircuitBreakerHalfOpen {
		// If we fail in half-open, go back to open
		cb.state = CircuitBreakerOpen
		cb.logger.Warn("circuit breaker reopened - service still failing")
	}
}

// recordSuccess records a success in the circuit breaker.
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == CircuitBreakerHalfOpen {
		cb.successCount++
	} else {
		// Reset failure count on success in closed state
		cb.failureCount = 0
	}
}

// ConnectPostgresWithRetry connects to PostgreSQL with retry logic and circuit breaker.
func ConnectPostgresWithRetry(ctx context.Context, logger *slog.Logger, dsn string, retryConfig RetryConfig, cb *CircuitBreaker) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	err = RetryWithBackoff(ctx, logger, retryConfig, func() error {
		return cb.Execute(func() error {
			var e error
			pool, e = pgxpool.New(ctx, dsn)
			if e != nil {
				return e
			}
			return pool.Ping(ctx)
		})
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

// ConnectRedisWithRetry connects to Redis with retry logic and circuit breaker.
func ConnectRedisWithRetry(ctx context.Context, logger *slog.Logger, addr, password string, retryConfig RetryConfig, cb *CircuitBreaker) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	var err error
	err = RetryWithBackoff(ctx, logger, retryConfig, func() error {
		return cb.Execute(func() error {
			return rdb.Ping(ctx).Err()
		})
	})

	if err != nil {
		rdb.Close()
		return nil, err
	}

	return rdb, nil
}

