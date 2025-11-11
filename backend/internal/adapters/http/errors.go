// Package http provides HTTP handlers for the pack optimizer API.
// This file contains error handling utilities for structured error responses.
package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// ErrorCode represents a machine-readable error code.
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"

	// Server errors (5xx)
	ErrCodeInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabaseError     ErrorCode = "DATABASE_ERROR"
	ErrCodeCalculationError  ErrorCode = "CALCULATION_ERROR"
)

// APIError represents a structured API error response.
type APIError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	StatusCode int                   `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails adds additional details to the error.
func (e *APIError) WithDetails(key string, value interface{}) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds a request ID to the error for tracing.
func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// NewAPIError creates a new API error with the given code, message, and status code.
func NewAPIError(code ErrorCode, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Common error constructors
var (
	ErrInvalidInput     = NewAPIError(ErrCodeInvalidInput, "Invalid input provided", http.StatusBadRequest)
	ErrValidationFailed = NewAPIError(ErrCodeValidationFailed, "Validation failed", http.StatusBadRequest)
	ErrInternalError    = NewAPIError(ErrCodeInternalError, "An internal error occurred", http.StatusInternalServerError)
	ErrDatabaseError    = NewAPIError(ErrCodeDatabaseError, "Database operation failed", http.StatusInternalServerError)
	ErrCalculationError = NewAPIError(ErrCodeCalculationError, "Calculation failed", http.StatusInternalServerError)
)

// ErrorHandler handles errors and writes structured error responses.
type ErrorHandler struct {
	logger      *slog.Logger
	development bool // If true, includes stack traces in errors
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(logger *slog.Logger, development bool) *ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ErrorHandler{
		logger:      logger,
		development: development,
	}
}

// HandleError writes a structured error response to the HTTP response writer.
func (h *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError

	// Check if it's already an APIError
	if apiError, ok := err.(*APIError); ok {
		apiErr = apiError
	} else {
		// Convert generic error to APIError
		apiErr = ErrInternalError
		apiErr.Message = err.Error()

		// Log the error with context using slog
		h.logger.Error(
			"unexpected error",
			"error", err.Error(),
			"path", r.URL.Path,
			"method", r.Method,
			"ip", r.RemoteAddr,
		)
	}

	// Add request ID from context if available
	if requestID := middleware.GetReqID(r.Context()); requestID != "" {
		apiErr = apiErr.WithRequestID(requestID)
	}

	// Add stack trace in development mode
	if h.development && apiErr.StatusCode >= 500 {
		stack := string(debug.Stack())
		apiErr = apiErr.WithDetails("stack_trace", strings.Split(stack, "\n"))
	}

	// Write error response
	h.writeErrorResponse(w, apiErr)
}

// HandleAPIError writes an APIError directly to the response.
func (h *ErrorHandler) HandleAPIError(w http.ResponseWriter, r *http.Request, apiErr *APIError) {
	// Add request ID from context if available
	if requestID := middleware.GetReqID(r.Context()); requestID != "" {
		apiErr = apiErr.WithRequestID(requestID)
	}

	// Log error with structured logging
	h.logger.Warn(
		"api error",
		"code", string(apiErr.Code),
		"message", apiErr.Message,
		"path", r.URL.Path,
		"method", r.Method,
		"status", apiErr.StatusCode,
		"details", apiErr.Details,
	)

	// Add stack trace in development mode
	if h.development && apiErr.StatusCode >= 500 {
		stack := string(debug.Stack())
		apiErr = apiErr.WithDetails("stack_trace", strings.Split(stack, "\n"))
	}

	h.writeErrorResponse(w, apiErr)
}

// writeErrorResponse writes the error response as JSON.
func (h *ErrorHandler) writeErrorResponse(w http.ResponseWriter, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode)

	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		h.logger.Error("failed to encode error response", "error", err)
	}
}

// RecoveryMiddleware recovers from panics and returns structured error responses.
func RecoveryMiddleware(errorHandler *ErrorHandler) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					var err error
					switch v := rec.(type) {
					case error:
						err = v
					case string:
						err = fmt.Errorf("%s", v)
					default:
						err = fmt.Errorf("%v", v)
					}

					errorHandler.logger.Error(
						"panic recovered",
						"error", err.Error(),
						"path", r.URL.Path,
						"method", r.Method,
						"stack", string(debug.Stack()),
					)

					errorHandler.HandleError(w, r, err)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware adds a request ID to the request context and response headers.
// Uses chi's middleware.RequestID for proper context handling.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

