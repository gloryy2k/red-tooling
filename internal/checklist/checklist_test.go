package checklist

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestAddAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "Recon", "DNS enumeration")
	Add(db, "eng-1", "Recon", "Port scanning")
	Add(db, "eng-1", "Exploit", "Initial access")

	items, err := List(db, "eng-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestCheckUncheck(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "Recon", "DNS enumeration")

	items, _ := List(db, "eng-1")
	id := items[0].ID

	if items[0].Done {
		t.Fatal("new item should not be done")
	}

	Check(db, id, "tester", 0)
	items, _ = List(db, "eng-1")
	if !items[0].Done {
		t.Fatal("checked item should be done")
	}

	Uncheck(db, id)
	items, _ = List(db, "eng-1")
	if items[0].Done {
		t.Fatal("unchecked item should not be done")
	}
}

func TestStats(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "Cat", "Item 1")
	Add(db, "eng-1", "Cat", "Item 2")
	Add(db, "eng-1", "Cat", "Item 3")

	items, _ := List(db, "eng-1")
	Check(db, items[0].ID, "op", 0)

	done, total := Stats(db, "eng-1")
	if total != 3 || done != 1 {
		t.Fatalf("expected 1/3, got %d/%d", done, total)
	}
}

func TestLoadPTES(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	count, err := LoadPTES(db, "eng-1")
	if err != nil {
		t.Fatalf("load ptes: %v", err)
	}
	if count != 27 {
		t.Fatalf("expected 27 PTES items, got %d", count)
	}

	items, _ := List(db, "eng-1")
	if len(items) != 27 {
		t.Fatalf("list should return 27, got %d", len(items))
	}
}
