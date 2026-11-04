
package storage

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	store := NewStorage()

	// Test 1: Non-existent key should return -2
	if ttl := store.TTL("unknown"); ttl != -2 {
		t.Errorf("expected -2, got %d", ttl)
	}

	// Test 2: Existing key without expiration should return -1
	store.Set("name", "Swarn")
	if ttl := store.TTL("name"); ttl != -1 {
		t.Errorf("expected -1, got %d", ttl)
	}

	// Test 3: Existing key with expiration should return positive TTL
	store.Set("city", "Delhi")
	store.Expire("city", 10)

	ttl := store.TTL("city")
	if ttl <= 0 || ttl > 10 {
		t.Errorf("expected TTL between 1 and 10, got %d", ttl)
	}

	// Test 4: Expired key should return -2 and be deleted
	store.Set("temp", "value")
	store.Expire("temp", 1)

	time.Sleep(1100 * time.Millisecond)

	if ttl := store.TTL("temp"); ttl != -2 {
		t.Errorf("expected -2 for expired key, got %d", ttl)
	}

	if _, exists := store.data["temp"]; exists {
		t.Error("expected expired key to be deleted")
	}
}
