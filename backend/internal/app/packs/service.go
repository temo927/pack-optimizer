package packs

import (
	"context"
)

type Repository interface {
	GetAllActive() ([]int, error)
	ReplaceActive(sizes []int) ([]int, error)
	CurrentVersion() (int64, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetActiveSizes(ctx context.Context) ([]int, error) {
	return s.repo.GetAllActive()
}

func (s *Service) ReplaceActive(ctx context.Context, sizes []int) ([]int, error) {
	return s.repo.ReplaceActive(sizes)
}


