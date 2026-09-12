package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

const Schema = `
CREATE TABLE IF NOT EXISTS engagements (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    client TEXT,
    scope_cidrs TEXT,
    scope_domains TEXT,
    start_date TEXT,
    end_date TEXT,
    status TEXT DEFAULT 'active',
    created_at TEXT DEFAULT (datetime('now')),
    metadata TEXT
);
CREATE TABLE IF NOT EXISTS operators (
    id TEXT PRIMARY KEY,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    role TEXT NOT NULL DEFAULT 'operator',
    api_key_hash TEXT,
    public_key TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    last_seen_at TEXT
);
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    name TEXT NOT NULL,
    source TEXT NOT NULL,
    operator TEXT REFERENCES operators(id),
    status TEXT DEFAULT 'active',
    started_at TEXT DEFAULT (datetime('now')),
    ended_at TEXT,
    metadata TEXT
);
CREATE TABLE IF NOT EXISTS evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    timestamp TEXT NOT NULL DEFAULT (datetime('now')),
    action TEXT NOT NULL,
    input TEXT,
    output TEXT,
    exit_code INTEGER,
    duration_ms INTEGER,
    cwd TEXT,
    tags TEXT,
    priority TEXT,
    verified TEXT,
    mitre TEXT,
    metadata TEXT,
    hash TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    operator_sig TEXT,
    is_deleted INTEGER DEFAULT 0,
    host TEXT DEFAULT ''
);
CREATE TABLE IF NOT EXISTS credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    username TEXT,
    secret_encrypted BLOB,
    secret_type TEXT,
    host TEXT,
    source_evidence_id INTEGER REFERENCES evidence(id),
    created_at TEXT DEFAULT (datetime('now')),
    notes TEXT
);
CREATE TABLE IF NOT EXISTS attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    evidence_id INTEGER REFERENCES evidence(id),
    filename TEXT NOT NULL,
    caption TEXT,
    filetype TEXT,
    size_bytes INTEGER,
    content BLOB,
    content_hash TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS findings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    title TEXT NOT NULL,
    description TEXT,
    priority TEXT,
    verified TEXT DEFAULT 'unverified',
    recommendation TEXT,
    evidence_ids TEXT,
    mitre TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    verified_by TEXT,
    verified_at TEXT,
    notes TEXT
);
CREATE TABLE IF NOT EXISTS checklist (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    category TEXT,
    item TEXT NOT NULL,
    done INTEGER DEFAULT 0,
    done_at TEXT,
    evidence_id INTEGER REFERENCES evidence(id),
    notes TEXT
);
CREATE TABLE IF NOT EXISTS scope_hosts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engagement_id TEXT NOT NULL REFERENCES engagements(id),
    host TEXT NOT NULL,
    tested INTEGER DEFAULT 0,
    tested_at TEXT,
    session_id TEXT
);
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL DEFAULT (datetime('now')),
    operator TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    detail TEXT,
    ip_address TEXT,
    request_hash TEXT
);
`

func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(Schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func SeedEngagement(t *testing.T, db *sql.DB, engID, name string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO engagements (id, name, status, created_at) VALUES (?, ?, 'active', datetime('now'))`,
		engID, name,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func SeedSession(t *testing.T, db *sql.DB, sessID, engID, name string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO sessions (id, engagement_id, name, source, status, started_at) VALUES (?, ?, ?, 'test', 'active', datetime('now'))`,
		sessID, engID, name,
	)
	if err != nil {
		t.Fatal(err)
	}
}
