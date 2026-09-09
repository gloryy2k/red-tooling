package engagement

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestCreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)

	if err := Create(db, "eng-1", "Test Engagement", "Acme Corp"); err != nil {
		t.Fatalf("create: %v", err)
	}

	eng, err := Get(db, "eng-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if eng.Name != "Test Engagement" {
		t.Fatalf("expected name 'Test Engagement', got %q", eng.Name)
	}
	if eng.Client != "Acme Corp" {
		t.Fatalf("expected client 'Acme Corp', got %q", eng.Client)
	}
	if eng.Status != "active" {
		t.Fatalf("expected status 'active', got %q", eng.Status)
	}
}

func TestList(t *testing.T) {
	db := testutil.NewTestDB(t)

	Create(db, "eng-1", "First", "Client A")
	Create(db, "eng-2", "Second", "Client B")

	list, err := List(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 engagements, got %d", len(list))
	}
}

func TestGetNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	_, err := Get(db, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent engagement")
	}
}
