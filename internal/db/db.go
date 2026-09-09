package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/crypto"
	_ "modernc.org/sqlite"
)

var (
	mu         sync.Mutex
	currentDB  *sql.DB
	currentEng string
	passphrase []byte
)

// IsUnlocked checks if the plaintext database file exists (engagement is unlocked).
func IsUnlocked(engName string) bool {
	plainPath := config.PlainDBPath(engName)
	_, err := os.Stat(plainPath)
	return err == nil
}

// OpenPlain opens an already-decrypted database (no passphrase needed).
func OpenPlain(engName string) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	if currentDB != nil && currentEng == engName {
		return currentDB, nil
	}

	plainPath := config.PlainDBPath(engName)
	if _, err := os.Stat(plainPath); err != nil {
		return nil, fmt.Errorf("database is locked. Run 'rt unlock' first")
	}

	db, err := sql.Open("sqlite", plainPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	currentDB = db
	currentEng = engName
	return db, nil
}

// Unlock decrypts the database and leaves the plaintext file for subsequent commands.
func Unlock(engName string, pass []byte) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	if currentDB != nil && currentEng == engName {
		return currentDB, nil
	}

	encPath := config.EncryptedDBPath(engName)
	plainPath := config.PlainDBPath(engName)

	// If already unlocked (plain file exists), just open it
	if _, err := os.Stat(plainPath); err == nil {
		return openSQLite(engName, plainPath)
	}

	// Decrypt
	if _, err := os.Stat(encPath); err != nil {
		return nil, fmt.Errorf("engagement %q not found", engName)
	}

	if err := crypto.DecryptFile(encPath, plainPath, pass); err != nil {
		return nil, fmt.Errorf("decrypt database: %w", err)
	}

	// Store passphrase for re-encryption on lock
	passphrase = make([]byte, len(pass))
	copy(passphrase, pass)

	// Save passphrase hash so we can re-encrypt on lock
	savePassphrase(engName, pass)

	return openSQLite(engName, plainPath)
}

// Lock re-encrypts the database and removes the plaintext file.
func Lock(engName string) error {
	mu.Lock()
	defer mu.Unlock()

	if currentDB != nil && currentEng == engName {
		currentDB.Close()
		currentDB = nil
	}

	plainPath := config.PlainDBPath(engName)
	encPath := config.EncryptedDBPath(engName)

	if _, err := os.Stat(plainPath); err != nil {
		return nil // already locked
	}

	pass := loadPassphrase(engName)
	if len(pass) == 0 {
		return fmt.Errorf("no passphrase cached, cannot re-encrypt. Use 'rt unlock' with passphrase first")
	}

	if err := crypto.EncryptFile(plainPath, encPath, pass); err != nil {
		return fmt.Errorf("encrypt database: %w", err)
	}

	os.Remove(plainPath)
	os.Remove(plainPath + "-wal")
	os.Remove(plainPath + "-shm")
	clearPassphrase(engName)
	crypto.ZeroBytes(pass)
	currentEng = ""

	return nil
}

// Close closes the current database connection (but does NOT lock).
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if currentDB == nil {
		return nil
	}

	err := currentDB.Close()
	currentDB = nil
	return err
}

// Current returns the currently open database, or nil.
func Current() *sql.DB {
	mu.Lock()
	defer mu.Unlock()
	return currentDB
}

// CurrentEngagement returns the name of the currently open engagement.
func CurrentEngagement() string {
	mu.Lock()
	defer mu.Unlock()
	return currentEng
}

// CreateNew creates a new encrypted engagement database with full schema.
func CreateNew(engName string, pass []byte) error {
	plainPath := config.PlainDBPath(engName)

	db, err := sql.Open("sqlite", plainPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		os.Remove(plainPath)
		return fmt.Errorf("migrate: %w", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("close after migrate: %w", err)
	}

	encPath := config.EncryptedDBPath(engName)
	if err := crypto.EncryptFile(plainPath, encPath, pass); err != nil {
		return fmt.Errorf("encrypt new database: %w", err)
	}
	os.Remove(plainPath)
	os.Remove(plainPath + "-wal")
	os.Remove(plainPath + "-shm")

	return nil
}

// ChangePassphrase re-encrypts the database with a new passphrase.
func ChangePassphrase(engName string, oldPass, newPass []byte) error {
	encPath := config.EncryptedDBPath(engName)
	plainPath := config.PlainDBPath(engName)

	// If unlocked, just re-encrypt with new pass
	if _, err := os.Stat(plainPath); err == nil {
		if err := crypto.EncryptFile(plainPath, encPath, newPass); err != nil {
			return fmt.Errorf("encrypt with new passphrase: %w", err)
		}
		savePassphrase(engName, newPass)
		return nil
	}

	// Locked: decrypt with old, re-encrypt with new
	if err := crypto.DecryptFile(encPath, plainPath, oldPass); err != nil {
		return fmt.Errorf("decrypt with old passphrase: %w", err)
	}

	if err := crypto.EncryptFile(plainPath, encPath, newPass); err != nil {
		os.Remove(plainPath)
		return fmt.Errorf("encrypt with new passphrase: %w", err)
	}

	os.Remove(plainPath)
	return nil
}

func openSQLite(engName, plainPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", plainPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	currentDB = db
	currentEng = engName
	return db, nil
}

// Passphrase caching — stored in a file so subsequent rt invocations can re-encrypt.
// The passphrase file is the raw passphrase XOR'd with a fixed pad (minimal obfuscation
// to avoid plaintext on disk). This is NOT secure against a determined attacker with disk
// access — the real protection is SQLCipher in production. For now, it enables the
// unlock-across-processes workflow.
func savePassphrase(engName string, pass []byte) {
	path := config.PlainDBPath(engName) + ".key"
	obf := make([]byte, len(pass))
	for i := range pass {
		obf[i] = pass[i] ^ 0xAA
	}
	os.WriteFile(path, obf, 0600)
}

func loadPassphrase(engName string) []byte {
	path := config.PlainDBPath(engName) + ".key"
	obf, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	pass := make([]byte, len(obf))
	for i := range obf {
		pass[i] = obf[i] ^ 0xAA
	}
	return pass
}

func clearPassphrase(engName string) {
	path := config.PlainDBPath(engName) + ".key"
	os.Remove(path)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
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
    is_deleted INTEGER DEFAULT 0
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

CREATE INDEX IF NOT EXISTS idx_evidence_session ON evidence(session_id);
CREATE INDEX IF NOT EXISTS idx_evidence_action ON evidence(action);
CREATE INDEX IF NOT EXISTS idx_evidence_tags ON evidence(tags);
CREATE INDEX IF NOT EXISTS idx_evidence_priority ON evidence(priority);
CREATE INDEX IF NOT EXISTS idx_evidence_timestamp ON evidence(timestamp);
CREATE INDEX IF NOT EXISTS idx_evidence_hash ON evidence(hash);
CREATE INDEX IF NOT EXISTS idx_credentials_engagement ON credentials(engagement_id);
CREATE INDEX IF NOT EXISTS idx_findings_engagement ON findings(engagement_id);
CREATE INDEX IF NOT EXISTS idx_scope_engagement ON scope_hosts(engagement_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_operator ON audit_log(operator);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_operators_engagement ON operators(engagement_id);
`
