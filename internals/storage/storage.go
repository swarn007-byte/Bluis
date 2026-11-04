package storage

import (
	"sync"
	"time"
)

type Storage struct {
	data map[string]string
	expires map[string]time.Time
	mu   sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		data: make(map[string]string),
		expires:make(map[string]time.Time),
	}
}

func (s *Storage) Set(key string, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _,exists:=s.data[key];exists{
		delete(s.expires, key)  //syntax is delete(map,key)
	}

	s.data[key] = val

}

func (s *Storage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, exists := s.data[key]
	return val, exists
}

func (s *Storage) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return false
	}

	delete(s.data, key)
	return true
}
func (s *Storage) Expire(key string, seconds int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; exists {
		s.expires[key] = time.Now().Add(time.Duration(seconds) * time.Second)
		return true
	}
	return false
}
