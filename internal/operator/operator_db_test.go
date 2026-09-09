package operator

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestAddAndAuthenticate(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	op, err := Add(db, "eng-1", "alice", "operator", "system")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if op.APIKey == "" {
		t.Fatal("expected API key")
	}
	if op.Role != "operator" {
		t.Fatalf("expected role operator, got %q", op.Role)
	}

	authed, err := Authenticate(db, "eng-1", op.APIKey)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if authed.ID != "alice" {
		t.Fatalf("expected alice, got %q", authed.ID)
	}
}

func TestAuthenticateInvalidKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "bob", "viewer", "system")

	_, err := Authenticate(db, "eng-1", "rt_key_invalid")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestRotateKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	op, _ := Add(db, "eng-1", "charlie", "lead", "system")
	oldKey := op.APIKey

	newKey, err := RotateKey(db, "eng-1", "charlie", "system")
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if newKey == oldKey {
		t.Fatal("rotated key should be different")
	}

	_, err = Authenticate(db, "eng-1", oldKey)
	if err == nil {
		t.Fatal("old key should no longer work")
	}

	_, err = Authenticate(db, "eng-1", newKey)
	if err != nil {
		t.Fatalf("new key should work: %v", err)
	}
}

func TestInvalidRole(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	_, err := Add(db, "eng-1", "dave", "superadmin", "system")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestList(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "alice", "lead", "system")
	Add(db, "eng-1", "bob", "operator", "system")

	ops, err := List(db, "eng-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ops) != 2 {
		t.Fatalf("expected 2 operators, got %d", len(ops))
	}
}
