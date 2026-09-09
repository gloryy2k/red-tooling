package evidence

import (
	"testing"

	"github.com/user/rt/internal/testutil"
)

func TestInsertAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test Engagement")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "test-session")

	ev, err := Insert(db, "sess-1", "command", "whoami", "root", 0, 100, "/tmp", []string{"recon"}, "high", "tester")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if ev.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if ev.Action != "command" {
		t.Fatalf("expected action 'command', got %q", ev.Action)
	}
	if ev.PrevHash != genesisHash {
		t.Fatalf("first entry should chain from genesis, got %q", ev.PrevHash)
	}

	got, err := Get(db, ev.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Input != "whoami" {
		t.Fatalf("expected input 'whoami', got %q", got.Input)
	}
}

func TestHashChain(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "chain-test")

	ev1, _ := Insert(db, "sess-1", "command", "cmd1", "out1", 0, 0, "", nil, "", "op")
	ev2, _ := Insert(db, "sess-1", "command", "cmd2", "out2", 0, 0, "", nil, "", "op")
	ev3, _ := Insert(db, "sess-1", "command", "cmd3", "out3", 0, 0, "", nil, "", "op")

	if ev1.PrevHash != genesisHash {
		t.Fatal("ev1 should chain from genesis")
	}
	if ev2.PrevHash != ev1.Hash {
		t.Fatal("ev2 should chain from ev1")
	}
	if ev3.PrevHash != ev2.Hash {
		t.Fatal("ev3 should chain from ev2")
	}
	if ev1.Hash == ev2.Hash || ev2.Hash == ev3.Hash {
		t.Fatal("each entry should have a unique hash")
	}
}

func TestSoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "del-test")

	ev, _ := Insert(db, "sess-1", "command", "secret", "sensitive data", 0, 0, "", nil, "", "op")

	if err := SoftDelete(db, []int64{ev.ID}, "op"); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	_, err := Get(db, ev.ID)
	if err == nil {
		t.Fatal("soft-deleted evidence should not be returned by Get")
	}
}

func TestRedact(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "redact-test")

	ev, _ := Insert(db, "sess-1", "command", "cat /etc/shadow", "root:$6$hash", 0, 0, "", nil, "", "op")

	if err := Redact(db, ev.ID, "contains credentials", "op"); err != nil {
		t.Fatalf("redact: %v", err)
	}

	got, err := Get(db, ev.ID)
	if err != nil {
		t.Fatalf("get after redact: %v", err)
	}
	if got.Output != "[REDACTED: contains credentials]" {
		t.Fatalf("expected redacted output, got %q", got.Output)
	}
}

func TestCountBySession(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.SeedEngagement(t, db, "eng-1", "Test")
	testutil.SeedSession(t, db, "sess-1", "eng-1", "count-test")

	Insert(db, "sess-1", "command", "cmd1", "out1", 0, 0, "", nil, "", "op")
	Insert(db, "sess-1", "command", "cmd2", "out2", 0, 0, "", nil, "", "op")
	Insert(db, "sess-1", "command", "cmd3", "out3", 0, 0, "", nil, "", "op")

	count := CountBySession(db, "sess-1")
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}
}
