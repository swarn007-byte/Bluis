
package storage

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistence(t *testing.T) {
	// Create first storage instance
	store1 := NewStorage()

	// Insert data
	store1.Set("name", "Swarn")
	store1.Set("city", "Delhi")

	// Create a temporary file for snapshot
	file, err := os.CreateTemp("", "bluis-*.rdb")
	if err != nil {
		t.Fatal(err)
	}
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	// Save first storage instance
	if err := store1.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Create a completely new storage instance
	store2 := NewStorage()

	// Load snapshot into second instance
	if err := store2.LoadFromFile(filename); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify restored data
	value, exists := store2.Get("name")
	if !exists {
		t.Fatal("expected key 'name' to exist")
	}

	if value != "Swarn" {
		t.Errorf("expected 'Swarn', got '%s'", value)
	}

	value, exists = store2.Get("city")
	if !exists {
		t.Fatal("expected key 'city' to exist")
	}

	if value != "Delhi" {
		t.Errorf("expected 'Delhi', got '%s'", value)
	}
}

func TestPersistenceTTL(t *testing.T) {
	store1 := NewStorage()

	store1.Set("name", "Swarn")
	store1.Set("city", "Delhi")
	store1.Expire("city", 100)

	store1.Set("gone", "value")
	store1.Expire("gone", -1)

	originalExpiry := store1.expires["city"]

	file, err := os.CreateTemp("", "bluis-*.rdb")
	if err != nil {
		t.Fatal(err)
	}
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	if err := store1.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	store2 := NewStorage()
	if err := store2.LoadFromFile(filename); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	restoredExpiry := store2.expires["city"]
	if !originalExpiry.Equal(restoredExpiry) {
		t.Errorf("expected expiry %v, got %v", originalExpiry, restoredExpiry)
	}

	ttl := store2.TTL("city")
	if ttl <= 0 || ttl > 100 {
		t.Errorf("expected TTL between 1 and 100, got %d", ttl)
	}

	value, exists := store2.Get("name")
	if !exists {
		t.Fatal("expected key 'name' to exist after reload")
	}
	if value != "Swarn" {
		t.Errorf("expected 'Swarn', got '%s'", value)
	}
	if ttl := store2.TTL("name"); ttl != -1 {
		t.Errorf("expected -1 for key without TTL, got %d", ttl)
	}
	if _, hasExpiry := store2.expires["name"]; hasExpiry {
		t.Error("expected no expiry entry for key without TTL")
	}

	if _, exists := store2.Get("gone"); exists {
		t.Error("expected expired key to be treated as expired after reload")
	}
	if ttl := store2.TTL("gone"); ttl != -2 {
		t.Errorf("expected -2 for expired key, got %d", ttl)
	}
}

func TestPersistenceLegacyFile(t *testing.T) {
	file, err := os.CreateTemp("", "bluis-legacy-*.rdb")
	if err != nil {
		t.Fatal(err)
	}
	filename := file.Name()

	if err := gob.NewEncoder(file).Encode(map[string]string{"name": "Swarn"}); err != nil {
		file.Close()
		os.Remove(filename)
		t.Fatalf("encoding legacy map failed: %v", err)
	}
	file.Close()
	defer os.Remove(filename)

	store := NewStorage()
	if err := store.LoadFromFile(filename); err != nil {
		t.Fatalf("LoadFromFile failed on legacy file: %v", err)
	}

	value, exists := store.Get("name")
	if !exists {
		t.Fatal("expected key 'name' to exist after loading legacy file")
	}
	if value != "Swarn" {
		t.Errorf("expected 'Swarn', got '%s'", value)
	}
	if ttl := store.TTL("name"); ttl != -1 {
		t.Errorf("expected -1 for key loaded from legacy file, got %d", ttl)
	}
}

func TestPersistenceFileErrors(t *testing.T) {
	store := NewStorage()

	if err := store.LoadFromFile(filepath.Join(t.TempDir(), "missing.rdb")); err == nil {
		t.Error("expected error when loading a missing file")
	} else if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error, got %v", err)
	}

	if err := store.SaveToFile(filepath.Join(t.TempDir(), "no-such-dir", "dump.rdb")); err == nil {
		t.Error("expected error when saving to a missing directory")
	}
}
