// Package http provides HTTP handlers for the pack optimizer API.
// This adapter implements the HTTP transport layer following hexagonal architecture principles.
package http

import (
	"encoding/json"
	"net/http"
	"context"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/temo/pack-optimizer/backend/internal/domain"
)

// PacksService defines the interface for pack size management operations.
// This abstraction allows the HTTP adapter to work with any implementation.
type PacksService interface {
	GetActiveSizes(ctx context.Context) ([]int, error)
	ReplaceActive(ctx context.Context, sizes []int) ([]int, error)
}

// packSvcAdapter is the HTTP adapter that bridges HTTP requests to domain services.
// It implements the adapter pattern from hexagonal architecture.
type packSvcAdapter struct {
	svc          PacksService      // Service for managing pack sizes
	calc         domain.Calculator // Service for calculating optimal pack distributions
	errorHandler *ErrorHandler     // Error handler for structured error responses
}

// NewRouter creates and configures a new HTTP router with all API endpoints.
// It sets up routes for pack management and calculation operations.
func NewRouter(packsSvc PacksService, calc domain.Calculator, errorHandler *ErrorHandler) chi.Router {
	r := chi.NewRouter()
	a := &packSvcAdapter{svc: packsSvc, calc: calc, errorHandler: errorHandler}
	
	// Root endpoint - returns API information
	r.Get("/", a.getRoot)
	
	// Health check endpoint for monitoring and load balancers
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	
	// Pack size management endpoints
	r.Get("/packs", a.getPacks)              // Retrieve current pack sizes
	r.Put("/packs", a.putPacks)              // Replace all pack sizes
	r.Delete("/packs/{size}", a.deletePack)  // Remove a specific pack size
	
	// Calculation endpoint
	r.Post("/calculate", a.postCalculate)    // Calculate optimal pack distribution
	
	return r
}

// getRoot returns API information and available endpoints.
func (a *packSvcAdapter) getRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        "Pack Optimizer API",
		"version":     "v1",
		"description": "API for calculating optimal pack distributions",
		"endpoints": map[string]string{
			"GET    /healthz":      "Health check",
			"GET    /packs":        "Get current pack sizes",
			"PUT    /packs":        "Replace all pack sizes",
			"DELETE /packs/{size}": "Remove a pack size",
			"POST   /calculate":    "Calculate optimal pack distribution",
		},
	})
}

// getPacks retrieves the current active pack sizes from the service.
// Returns a JSON response with the list of pack sizes.
func (a *packSvcAdapter) getPacks(w http.ResponseWriter, r *http.Request) {
	sizes, err := a.svc.GetActiveSizes(r.Context())
	if err != nil {
		a.errorHandler.HandleError(w, r, ErrDatabaseError.WithDetails("operation", "get_pack_sizes"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

// deletePack removes a specific pack size from the active set.
// It extracts the size from the URL path parameter, filters it out from current sizes,
// and updates the pack sizes. If the size doesn't exist, returns current sizes unchanged.
func (a *packSvcAdapter) deletePack(w http.ResponseWriter, r *http.Request) {
	// Extract size from URL path parameter
	sizeStr := chi.URLParam(r, "size")
	if sizeStr == "" {
		a.errorHandler.HandleAPIError(w, r, ErrInvalidInput.WithDetails("field", "size").WithDetails("reason", "size parameter is required"))
		return
	}
	
	// Validate and parse the size parameter
	val, err := strconv.Atoi(sizeStr)
	if err != nil || val <= 0 {
		a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.WithDetails("field", "size").WithDetails("value", sizeStr).WithDetails("reason", "must be a positive integer"))
		return
	}
	
	// Get current pack sizes
	curr, err := a.svc.GetActiveSizes(r.Context())
	if err != nil {
		a.errorHandler.HandleError(w, r, ErrDatabaseError.WithDetails("operation", "get_pack_sizes"))
		return
	}
	
	// Filter out the size to be deleted
	next := make([]int, 0, len(curr))
	for _, s := range curr {
		if s != val {
			next = append(next, s)
		}
	}
	
	// If nothing changed (size not found), return current sizes
	if len(next) == len(curr) {
		writeJSON(w, http.StatusOK, map[string]any{"sizes": curr})
		return
	}
	
	// Update pack sizes with the filtered list
	sizes, err := a.svc.ReplaceActive(r.Context(), next)
	if err != nil {
		a.errorHandler.HandleError(w, r, ErrDatabaseError.WithDetails("operation", "replace_pack_sizes"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

// putPacksReq represents the request body for updating pack sizes.
type putPacksReq struct {
	Sizes []int `json:"sizes"`
}

// putPacks replaces all pack sizes with a new set provided in the request body.
// Validates that all sizes are positive integers and within the maximum limit (10,000).
// Allows empty arrays - validation for zero sizes happens at calculation time.
func (a *packSvcAdapter) putPacks(w http.ResponseWriter, r *http.Request) {
	var req putPacksReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.errorHandler.HandleAPIError(w, r, ErrInvalidInput.WithDetails("field", "body").WithDetails("reason", "invalid JSON format"))
		return
	}
	
	// Allow empty arrays - validation happens at calculation time
	// Validate pack sizes: must be positive and <= 10,000
	const maxPackSize = 10_000
	for i, s := range req.Sizes {
		if s <= 0 {
			a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.
				WithDetails("field", "sizes").
				WithDetails("index", i).
				WithDetails("value", s).
				WithDetails("reason", "pack sizes must be positive"))
			return
		}
		if s > maxPackSize {
			a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.
				WithDetails("field", "sizes").
				WithDetails("index", i).
				WithDetails("value", s).
				WithDetails("reason", "pack sizes cannot exceed 10,000 items"))
			return
		}
	}
	
	// Replace all pack sizes with the new set
	sizes, err := a.svc.ReplaceActive(r.Context(), req.Sizes)
	if err != nil {
		a.errorHandler.HandleError(w, r, ErrDatabaseError.WithDetails("operation", "replace_pack_sizes"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

// calcReq represents the request body for pack calculation.
type calcReq struct {
	Amount int   `json:"amount"`           // Number of items to fulfill
	Sizes  []int `json:"sizes,omitempty"`  // Optional custom pack sizes (uses active if empty)
}

// postCalculate computes the optimal pack distribution for a given amount.
// Validates the amount is positive and within limits (1,000,000).
// If no custom sizes are provided, uses the active pack sizes from the service.
// Returns a breakdown showing how many packs of each size are needed.
func (a *packSvcAdapter) postCalculate(w http.ResponseWriter, r *http.Request) {
	var req calcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.errorHandler.HandleAPIError(w, r, ErrInvalidInput.WithDetails("field", "body").WithDetails("reason", "invalid JSON format"))
		return
	}
	
	// Validate amount is positive
	if req.Amount <= 0 {
		a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.WithDetails("field", "amount").WithDetails("value", req.Amount).WithDetails("reason", "amount must be positive"))
		return
	}
	
	// Validate amount doesn't exceed maximum limit
	const maxAmount = 1_000_000
	if req.Amount > maxAmount {
		a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.WithDetails("field", "amount").WithDetails("value", req.Amount).WithDetails("reason", "amount cannot exceed 1,000,000 items"))
		return
	}
	
	// Use custom sizes if provided, otherwise fetch active sizes
	sizes := req.Sizes
	if len(sizes) == 0 {
		var err error
		sizes, err = a.svc.GetActiveSizes(r.Context())
		if err != nil {
			a.errorHandler.HandleError(w, r, ErrDatabaseError.WithDetails("operation", "get_pack_sizes"))
			return
		}
	}
	
	// Ensure at least one pack size is configured
	if len(sizes) == 0 {
		a.errorHandler.HandleAPIError(w, r, ErrValidationFailed.WithDetails("field", "sizes").WithDetails("reason", "no pack sizes configured"))
		return
	}
	
	// Perform the calculation
	res, err := a.calc.Compute(r.Context(), req.Amount, sizes)
	if err != nil {
		a.errorHandler.HandleError(w, r, ErrCalculationError.WithDetails("amount", req.Amount))
		return
	}
	
	// Format breakdown with descending sizes for better readability
	breakdown := map[string]int{}
	keys := make([]int, 0, len(res.Breakdown))
	for s := range res.Breakdown {
		keys = append(keys, s)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	for _, s := range keys {
		breakdown[strconv.Itoa(s)] = res.Breakdown[s]
	}
	
	// Return calculation result
	writeJSON(w, http.StatusOK, map[string]any{
		"amount":     req.Amount,
		"totalItems": res.TotalItems,
		"totalPacks": res.TotalPacks,
		"breakdown":  breakdown,
		"overage":    res.Overage,
	})
}

// writeJSON is a helper function to write JSON responses with proper headers.
// Sets Content-Type header and writes the response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
