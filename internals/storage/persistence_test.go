
package storage

import (
	"os"
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
