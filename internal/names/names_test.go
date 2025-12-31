package names

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	name := Generate()

	// Verify format: adjective_surname
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		t.Errorf("expected name with format 'adjective_surname', got %q", name)
	}

	// Verify non-empty parts
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("expected non-empty adjective and surname, got %q", name)
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	// Generate multiple names and verify we get variety
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		names[Generate()] = true
	}

	// With ~25k combinations, 100 generations should yield mostly unique names
	if len(names) < 50 {
		t.Errorf("expected more unique names, got only %d unique out of 100", len(names))
	}
}

func TestGenerateUnique(t *testing.T) {
	existing := make(map[string]bool)
	existsFn := func(name string) bool {
		return existing[name]
	}

	// Generate several unique names
	for i := 0; i < 10; i++ {
		name, err := GenerateUnique(existsFn, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existing[name] {
			t.Errorf("generated duplicate name: %s", name)
		}
		existing[name] = true
	}
}

func TestGenerateUnique_AllExist(t *testing.T) {
	// Everything exists
	existsFn := func(name string) bool {
		return true
	}

	_, err := GenerateUnique(existsFn, 10)
	if err == nil {
		t.Error("expected error when all names exist")
	}
}

func TestGenerateUnique_DefaultMaxAttempts(t *testing.T) {
	existsFn := func(name string) bool {
		return true
	}

	// Pass 0 to use default max attempts
	_, err := GenerateUnique(existsFn, 0)
	if err == nil {
		t.Error("expected error when all names exist")
	}
}
