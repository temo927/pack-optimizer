package http

import (
	"encoding/json"
	"net/http"
	"context"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/temo/pack-optimizer/backend/internal/domain"
)

type PacksService interface {
	GetActiveSizes(ctx context.Context) ([]int, error)
	ReplaceActive(ctx context.Context, sizes []int) ([]int, error)
}

type packSvcAdapter struct {
	svc  PacksService
	calc domain.Calculator
}

func NewRouter(packsSvc PacksService, calc domain.Calculator) chi.Router {
	r := chi.NewRouter()
	a := &packSvcAdapter{svc: packsSvc, calc: calc}
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/packs", a.getPacks)
	r.Put("/packs", a.putPacks)
	r.Delete("/packs/{size}", a.deletePack)
	r.Post("/calculate", a.postCalculate)
	return r
}

func (a *packSvcAdapter) getPacks(w http.ResponseWriter, r *http.Request) {
	sizes, err := a.svc.GetActiveSizes(r.Context())
	if err != nil {
		http.Error(w, "failed to load sizes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

func (a *packSvcAdapter) deletePack(w http.ResponseWriter, r *http.Request) {
	sizeStr := chi.URLParam(r, "size")
	if sizeStr == "" {
		http.Error(w, "size required", http.StatusBadRequest)
		return
	}
	val, err := strconv.Atoi(sizeStr)
	if err != nil || val <= 0 {
		http.Error(w, "invalid size", http.StatusBadRequest)
		return
	}
	curr, err := a.svc.GetActiveSizes(r.Context())
	if err != nil {
		http.Error(w, "failed to load sizes", http.StatusInternalServerError)
		return
	}
	// Prevent deleting the last pack size
	if len(curr) <= 1 {
		http.Error(w, "at least one pack size must remain", http.StatusBadRequest)
		return
	}
	next := make([]int, 0, len(curr))
	for _, s := range curr {
		if s != val {
			next = append(next, s)
		}
	}
	if len(next) == len(curr) {
		// nothing changed; still return current
		writeJSON(w, http.StatusOK, map[string]any{"sizes": curr})
		return
	}
	sizes, err := a.svc.ReplaceActive(r.Context(), next)
	if err != nil {
		http.Error(w, "failed to update sizes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

type putPacksReq struct {
	Sizes []int `json:"sizes"`
}

func (a *packSvcAdapter) putPacks(w http.ResponseWriter, r *http.Request) {
	var req putPacksReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Sizes) == 0 {
		http.Error(w, "sizes required", http.StatusBadRequest)
		return
	}
	// Validate pack sizes: must be positive and <= 10,000
	const maxPackSize = 10_000
	for _, s := range req.Sizes {
		if s <= 0 {
			http.Error(w, "pack sizes must be positive", http.StatusBadRequest)
			return
		}
		if s > maxPackSize {
			http.Error(w, "pack sizes cannot exceed 10,000 items", http.StatusBadRequest)
			return
		}
	}
	sizes, err := a.svc.ReplaceActive(r.Context(), req.Sizes)
	if err != nil {
		log.Error().Err(err).Msg("replace sizes failed")
		http.Error(w, "failed to update sizes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sizes": sizes})
}

type calcReq struct {
	Amount int   `json:"amount"`
	Sizes  []int `json:"sizes,omitempty"`
}

func (a *packSvcAdapter) postCalculate(w http.ResponseWriter, r *http.Request) {
	var req calcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}
	sizes := req.Sizes
	if len(sizes) == 0 {
		var err error
		sizes, err = a.svc.GetActiveSizes(r.Context())
		if err != nil {
			http.Error(w, "failed to load sizes", http.StatusInternalServerError)
			return
		}
	}
	res, err := a.calc.Compute(r.Context(), req.Amount, sizes)
	if err != nil {
		http.Error(w, "calculation failed", http.StatusInternalServerError)
		return
	}
	breakdown := map[string]int{}
	// present breakdown with descending sizes for readability
	keys := make([]int, 0, len(res.Breakdown))
	for s := range res.Breakdown {
		keys = append(keys, s)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	for _, s := range keys {
		breakdown[strconv.Itoa(s)] = res.Breakdown[s]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"amount":     req.Amount,
		"totalItems": res.TotalItems,
		"totalPacks": res.TotalPacks,
		"breakdown":  breakdown,
		"overage":    res.Overage,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}


