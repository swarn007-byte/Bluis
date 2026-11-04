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
	s.mu.Lock()
	defer s.mu.Unlock()

	val, exists := s.data[key]
	if !exists {
		return "", false
	}

	expiry, hasExpiry := s.expires[key]
	if hasExpiry && time.Now().After(expiry) {
		delete(s.data, key)
		delete(s.expires, key)
		return "", false
	}

	return val, true
}

func (s *Storage) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return false
	}

	delete(s.data, key)
	delete(s.expires, key)

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
func (s *Storage) TTL(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.data[key]
	if !exists {
		return -2
	}

	expiry, hasExpiry := s.expires[key]
	if !hasExpiry {
		return -1
	}

	if time.Until(expiry) <= 0 {
		delete(s.data, key)
		delete(s.expires, key)
		return -2
	}

	return int(time.Until(expiry).Seconds())
}

func (s *Storage) StartCleanup(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			s.mu.Lock()
			for key, expiry := range s.expires {
				if now.After(expiry) {
					delete(s.data, key)
					delete(s.expires, key)
				}
			}
			s.mu.Unlock()
		case <-stop:
			return
		}
	}
}
