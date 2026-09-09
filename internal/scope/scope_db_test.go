package scope

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestAddAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	count, err := Add(db, "eng-1", "10.0.0.1,10.0.0.2,10.0.0.3", "tester")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 hosts added, got %d", count)
	}

	hosts, err := List(db, "eng-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(hosts))
	}
}

func TestMarkTested(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "10.0.0.1,10.0.0.2", "tester")

	err := MarkTested(db, "eng-1", "10.0.0.1", "sess-1", "tester")
	if err != nil {
		t.Fatalf("mark tested: %v", err)
	}

	total, tested := Stats(db, "eng-1")
	if total != 2 || tested != 1 {
		t.Fatalf("expected 2 total, 1 tested; got %d/%d", total, tested)
	}
}

func TestMarkTestedNotInScope(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	err := MarkTested(db, "eng-1", "192.168.1.1", "", "tester")
	if err == nil {
		t.Fatal("expected error for host not in scope")
	}
}

func TestUntested(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Add(db, "eng-1", "10.0.0.1,10.0.0.2,10.0.0.3", "tester")
	MarkTested(db, "eng-1", "10.0.0.1", "", "tester")

	untested, err := Untested(db, "eng-1")
	if err != nil {
		t.Fatalf("untested: %v", err)
	}
	if len(untested) != 2 {
		t.Fatalf("expected 2 untested, got %d", len(untested))
	}
}
