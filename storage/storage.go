package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("receipt not found")

type Storage interface {
	SavePoints(ctx context.Context, points int) (string, error)
	GetPoints(ctx context.Context, id string) (int, error)
}

type InMemoryStorage struct {
	mu     sync.RWMutex
	points map[string]int
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		points: make(map[string]int),
	}
}

func (s *InMemoryStorage) SavePoints(ctx context.Context, points int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	s.points[id] = points
	return id, nil
}

func (s *InMemoryStorage) GetPoints(ctx context.Context, id string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, exists := s.points[id]
	if !exists {
		return 0, ErrNotFound
	}
	return points, nil
}
