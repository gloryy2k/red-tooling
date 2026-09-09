package operator

import (
	"testing"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		role     string
		resource string
		want     bool
	}{
		{"lead", "overview", true},
		{"lead", "operators", true},
		{"lead", "audit", true},
		{"operator", "overview", true},
		{"operator", "operators", false},
		{"operator", "audit", false},
		{"reviewer", "overview", true},
		{"reviewer", "verify", true},
		{"reviewer", "creds", false},
		{"viewer", "overview", true},
		{"viewer", "creds", false},
		{"viewer", "audit", false},
		{"viewer", "evidence", false},
		{"unknown", "overview", false},
	}

	for _, tt := range tests {
		t.Run(tt.role+"_"+tt.resource, func(t *testing.T) {
			got := HasPermission(tt.role, tt.resource)
			if got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.resource, got, tt.want)
			}
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	h1 := hashAPIKey("rt_key_test123")
	h2 := hashAPIKey("rt_key_test123")
	if h1 != h2 {
		t.Fatal("same key should produce same hash")
	}

	h3 := hashAPIKey("rt_key_different")
	if h1 == h3 {
		t.Fatal("different keys should produce different hashes")
	}

	if len(h1) != 64 {
		t.Fatalf("expected 64 char hex hash, got %d", len(h1))
	}
}
