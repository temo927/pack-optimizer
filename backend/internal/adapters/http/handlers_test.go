package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/temo/pack-optimizer/backend/internal/domain"
)

// Test helpers to reduce duplication

// newTestErrorHandler creates an error handler for testing.
func newTestErrorHandler() *ErrorHandler {
	return NewErrorHandler(slog.Default(), true)
}

// newTestRouter creates a router with mocked services for testing.
func newTestRouter(packsSvc domain.PacksService, calc domain.Calculator) chi.Router {
	return NewRouter(packsSvc, calc, newTestErrorHandler())
}

// newTestRequest creates an HTTP test request with JSON body.
func newTestRequest(method, path string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	return req
}

// mockPacksService implements domain.PacksService for testing.
type mockPacksService struct {
	sizes []int
	err   error
}

func (m *mockPacksService) GetActiveSizes(ctx context.Context) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sizes, nil
}

func (m *mockPacksService) ReplaceActive(ctx context.Context, sizes []int) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.sizes = sizes
	return sizes, nil
}

// mockCalculator implements domain.Calculator for testing.
type mockCalculator struct {
	result domain.CalculationResult
	err    error
}

func (m *mockCalculator) Compute(ctx context.Context, amount int, sizes []int) (domain.CalculationResult, error) {
	if m.err != nil {
		return domain.CalculationResult{}, m.err
	}
	return m.result, nil
}

func TestGetPacks(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500, 1000}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	req := newTestRequest("GET", "/packs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Sizes []int `json:"sizes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Sizes) != 3 {
		t.Errorf("Expected 3 sizes, got %d", len(response.Sizes))
	}
}

func TestPutPacks(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	body := map[string][]int{"sizes": {250, 500, 1000}}
	req := newTestRequest("PUT", "/packs", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if len(svc.sizes) != 3 {
		t.Errorf("Expected 3 sizes after update, got %d", len(svc.sizes))
	}
}

func TestPutPacks_InvalidInput(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	// Test with empty sizes - should succeed now
	body := map[string][]int{"sizes": {}}
	req := newTestRequest("PUT", "/packs", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty sizes, got %d", w.Code)
	}

	// Test with size exceeding 10,000
	body = map[string][]int{"sizes": {250, 500, 15000}}
	req = newTestRequest("PUT", "/packs", body)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for size > 10000, got %d", w.Code)
	}

	// Test with negative size
	body = map[string][]int{"sizes": {250, -100}}
	req = newTestRequest("PUT", "/packs", body)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for negative size, got %d", w.Code)
	}
}

func TestDeletePack(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500, 1000}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	req := newTestRequest("DELETE", "/packs/500", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify size was removed
	sizes, _ := svc.GetActiveSizes(context.Background())
	found := false
	for _, s := range sizes {
		if s == 500 {
			found = true
			break
		}
	}
	if found {
		t.Errorf("Size 500 should have been removed")
	}
}

func TestDeletePack_LastOne(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	req := newTestRequest("DELETE", "/packs/250", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed - we now allow deleting all pack sizes
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when deleting last pack size, got %d", w.Code)
	}

	// Verify size was removed
	sizes, _ := svc.GetActiveSizes(context.Background())
	if len(sizes) != 0 {
		t.Errorf("Expected empty sizes after deleting last one, got %v", sizes)
	}
}

func TestCalculate_NoPackSizes(t *testing.T) {
	svc := &mockPacksService{sizes: []int{}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	body := map[string]int{"amount": 100}
	req := newTestRequest("POST", "/calculate", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for calculation with no pack sizes, got %d", w.Code)
	}

	// Parse JSON error response
	var errResp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Errorf("Expected JSON error response, got %q", w.Body.String())
		return
	}

	if errResp.Code != ErrCodeValidationFailed {
		t.Errorf("Expected error code VALIDATION_FAILED, got %s", errResp.Code)
	}

	if errResp.Details["reason"] != "no pack sizes configured" {
		t.Errorf("Expected error reason 'no pack sizes configured', got %v", errResp.Details)
	}
}

func TestCalculate(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500, 1000}}
	calc := &mockCalculator{
		result: domain.CalculationResult{
			Amount:     263,
			TotalItems: 500,
			Overage:    237,
			TotalPacks: 1,
			Breakdown:  map[int]int{500: 1},
		},
	}
	router := newTestRouter(svc, calc)

	body := map[string]int{"amount": 263}
	req := newTestRequest("POST", "/calculate", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response domain.CalculationResult
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Amount != 263 {
		t.Errorf("Expected amount 263, got %d", response.Amount)
	}
	if response.TotalItems != 500 {
		t.Errorf("Expected totalItems 500, got %d", response.TotalItems)
	}
}

func TestCalculate_InvalidAmount(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500}}
	calc := &mockCalculator{}
	router := newTestRouter(svc, calc)

	// Test with zero amount
	body := map[string]int{"amount": 0}
	req := newTestRequest("POST", "/calculate", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for zero amount, got %d", w.Code)
	}

	// Test with negative amount
	body = map[string]int{"amount": -1}
	req = newTestRequest("POST", "/calculate", body)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for negative amount, got %d", w.Code)
	}

	// Test with amount exceeding 1 million
	body = map[string]int{"amount": 1_000_001}
	req = newTestRequest("POST", "/calculate", body)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for amount > 1 million, got %d", w.Code)
	}
}

func TestCalculate_WithCustomSizes(t *testing.T) {
	svc := &mockPacksService{sizes: []int{250, 500}}
	calc := &mockCalculator{
		result: domain.CalculationResult{
			Amount:     500000,
			TotalItems: 500000,
			Overage:    0,
			TotalPacks: 9438,
			Breakdown:  map[int]int{23: 2, 31: 7, 53: 9429},
		},
	}
	router := newTestRouter(svc, calc)

	body := map[string]interface{}{
		"amount": 500000,
		"sizes":  []int{23, 31, 53},
	}
	req := newTestRequest("POST", "/calculate", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

