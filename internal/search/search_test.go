package search

import (
	"strings"
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestQuerySafety(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr string
	}{
		{"drop blocked", "DROP TABLE evidence", "only SELECT"},
		{"delete blocked", " DELETE FROM evidence", "only SELECT"},
		{"insert blocked", "INSERT INTO evidence VALUES (1)", "only SELECT"},
		{"update blocked", "UPDATE evidence SET output = 'x'", "only SELECT"},
		{"alter blocked", "ALTER TABLE evidence ADD col TEXT", "only SELECT"},
		{"create blocked", "CREATE TABLE evil (id INT)", "only SELECT"},
		{"truncate blocked", "TRUNCATE TABLE evidence", "only SELECT"},
		{"select with drop", "SELECT * FROM t; DROP TABLE t", "disallowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Query(nil, tt.query)
			if err == nil {
				t.Fatal("expected error for dangerous query")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestQueryAllowsIsDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	results, err := Query(db, "SELECT COUNT(*) as cnt FROM evidence WHERE is_deleted = 0")
	if err != nil {
		t.Fatalf("query with is_deleted should work: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 row, got %d", len(results))
	}
}

func TestQueryValidSelect(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")

	results, err := Query(db, "SELECT id, name FROM engagements")
	if err != nil {
		t.Fatalf("valid select should work: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 engagement, got %d", len(results))
	}
}

func TestSearchEmpty(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "test")

	results, err := Search(db, "eng-1", "nonexistent", "", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
