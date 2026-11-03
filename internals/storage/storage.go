package storage

import "sync"

type Storage struct {
	data map[string]string
	mu   sync.RWMutex
}

// initializing
func NewStorage() *Storage {
	return &Storage{
		data: make(map[string]string),
	}
}

// set key value pair.
func (s *Storage) Set(key string, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = val
}

//get key value pair.

func (s *Storage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, exists := s.data[key]
	return val, exists
}

// delete particulr key value pair.
// reports whether the key existed.
func (s *Storage) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return false
	}

	delete(s.data, key)
	return true
}
