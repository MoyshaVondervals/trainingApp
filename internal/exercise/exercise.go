package exercise

import (
	"errors"
	"strings"
	"sync"
	"unicode/utf8"
)

var ErrNotFound = errors.New("exercise not found")

type Exercise struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	MuscleGroup string `json:"muscle_group"`
	Description string `json:"description,omitempty"`
}

// Store — временное хранилище в памяти. На шаге 2 его заменит репозиторий на Postgres.
type Store struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Exercise
}

func NewStore() *Store {
	return &Store{items: make(map[int64]Exercise)}
}

func (s *Store) Create(e Exercise) Exercise {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	e.ID = s.nextID
	s.items[e.ID] = e
	return e
}

func (s *Store) Get(id int64) (Exercise, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.items[id]
	if !ok {
		return Exercise{}, ErrNotFound
	}
	return e, nil
}

func (s *Store) List(limit int) []Exercise {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Exercise, 0, len(s.items))
	for _, e := range s.items {
		if len(out) == limit {
			break
		}
		out = append(out, e)
	}
	return out
}

func Validate(name, muscleGroup string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(name) > 100 {
		return errors.New("name must be at most 100 characters")
	}
	if strings.TrimSpace(muscleGroup) == "" {
		return errors.New("muscle_group is required")
	}
	if utf8.RuneCountInString(muscleGroup) > 500 {
		return errors.New("muscle_group must be at most 500 characters")
	}
	return nil
}
