package findings

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestCreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	f, err := Create(db, "eng-1", "SQLi in login", "POST /login vulnerable", "high", "tester", []int64{1, 2}, []string{"T1190"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if f.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := Get(db, f.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "SQLi in login" {
		t.Fatalf("expected title 'SQLi in login', got %q", got.Title)
	}
	if got.Verified != "unverified" {
		t.Fatalf("new finding should be unverified, got %q", got.Verified)
	}
}

func TestVerify(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	f, _ := Create(db, "eng-1", "Test Finding", "", "medium", "op", nil, nil)

	if err := Verify(db, f.ID, "confirmed", "reviewer", "verified in retest"); err != nil {
		t.Fatalf("verify: %v", err)
	}

	got, _ := Get(db, f.ID)
	if got.Verified != "confirmed" {
		t.Fatalf("expected confirmed, got %q", got.Verified)
	}
	if got.VerifiedBy != "reviewer" {
		t.Fatalf("expected verified by reviewer, got %q", got.VerifiedBy)
	}
}

func TestMerge(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	f1, _ := Create(db, "eng-1", "SQLi #1", "desc1", "high", "op", []int64{1}, []string{"T1190"})
	f2, _ := Create(db, "eng-1", "SQLi #2", "desc2", "critical", "op", []int64{2}, []string{"T1190"})

	merged, err := Merge(db, []int64{f1.ID, f2.ID}, "Merged SQLi", "op")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	if merged.Priority != "critical" {
		t.Fatalf("merged should keep highest priority (critical), got %q", merged.Priority)
	}

	_, err = Get(db, f1.ID)
	if err == nil {
		t.Fatal("original f1 should be deleted after merge")
	}
	_, err = Get(db, f2.ID)
	if err == nil {
		t.Fatal("original f2 should be deleted after merge")
	}
}

func TestSetRecommendation(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	f, _ := Create(db, "eng-1", "Weak TLS", "", "medium", "op", nil, nil)
	SetRecommendation(db, f.ID, "Upgrade to TLS 1.3", "op")

	got, _ := Get(db, f.ID)
	if got.Recommendation != "Upgrade to TLS 1.3" {
		t.Fatalf("expected recommendation, got %q", got.Recommendation)
	}
}

func TestList(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	Create(db, "eng-1", "Critical RCE", "", "critical", "op", nil, nil)
	Create(db, "eng-1", "Low info disc", "", "low", "op", nil, nil)
	Create(db, "eng-1", "High SQLi", "", "high", "op", nil, nil)

	list, err := List(db, "eng-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(list))
	}
	if list[0].Priority != "critical" {
		t.Fatalf("first finding should be critical (sorted), got %q", list[0].Priority)
	}
}

func TestCountByStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	f1, _ := Create(db, "eng-1", "F1", "", "high", "op", nil, nil)
	f2, _ := Create(db, "eng-1", "F2", "", "medium", "op", nil, nil)
	Create(db, "eng-1", "F3", "", "low", "op", nil, nil)

	Verify(db, f1.ID, "confirmed", "reviewer", "")
	Verify(db, f2.ID, "false-positive", "reviewer", "")

	uv, cf, fp := CountByStatus(db, "eng-1")
	if uv != 1 || cf != 1 || fp != 1 {
		t.Fatalf("expected 1/1/1, got %d/%d/%d", uv, cf, fp)
	}
}
