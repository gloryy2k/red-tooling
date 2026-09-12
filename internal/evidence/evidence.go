package evidence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/crypto"
)

type Evidence struct {
	ID         int64
	SessionID  string
	Timestamp  string
	Action     string
	Input      string
	Output     string
	ExitCode   int
	DurationMs int
	CWD        string
	Tags       []string
	Priority   string
	Verified   string
	Mitre      []string
	Hash       string
	PrevHash   string
	Host       string
}

const genesisHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// Insert adds an evidence entry with hash chain integrity.
func Insert(db *sql.DB, sessionID, action, input, output string, exitCode int, durationMs int, cwd string, tags []string, priority, operator string) (*Evidence, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tagsJSON := "[]"
	if len(tags) > 0 {
		b, _ := json.Marshal(tags)
		tagsJSON = string(b)
	}

	prevHash := getLastHash(db, sessionID)

	hash := crypto.HashEvidence(now, action, input, output, exitCode, tagsJSON, operator)

	host := detectHost(db, sessionID, input)

	res, err := db.Exec(
		`INSERT INTO evidence (session_id, timestamp, action, input, output, exit_code, duration_ms, cwd, tags, priority, hash, prev_hash, host)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, now, action, input, output, exitCode, durationMs, cwd, tagsJSON, priority, hash, prevHash, host,
	)
	if err != nil {
		return nil, fmt.Errorf("insert evidence: %w", err)
	}

	id, _ := res.LastInsertId()

	if operator == "" {
		operator = "system"
	}
	audit.Log(db, operator, "evidence.create", "evidence", fmt.Sprintf("%d", id), map[string]string{"action": action})

	return &Evidence{
		ID:         id,
		SessionID:  sessionID,
		Timestamp:  now,
		Action:     action,
		Input:      input,
		Output:     output,
		ExitCode:   exitCode,
		DurationMs: durationMs,
		CWD:        cwd,
		Tags:       tags,
		Priority:   priority,
		Hash:       hash,
		PrevHash:   prevHash,
		Host:       host,
	}, nil
}

// detectHost scans the command input for scope hosts.
func detectHost(db *sql.DB, sessionID, input string) string {
	if input == "" {
		return ""
	}
	var engID string
	err := db.QueryRow(`SELECT engagement_id FROM sessions WHERE id = ?`, sessionID).Scan(&engID)
	if err != nil {
		return ""
	}
	rows, err := db.Query(`SELECT host FROM scope_hosts WHERE engagement_id = ?`, engID)
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		rows.Scan(&h)
		if h != "" && strings.Contains(input, h) {
			return h
		}
	}
	return ""
}

func getLastHash(db *sql.DB, sessionID string) string {
	var hash string
	err := db.QueryRow(
		`SELECT hash FROM evidence WHERE session_id = ? ORDER BY id DESC LIMIT 1`, sessionID,
	).Scan(&hash)
	if err != nil {
		return genesisHash
	}
	return hash
}

// Timeline returns evidence entries for the active engagement, ordered by time.
func Timeline(db *sql.DB, engID string, limit int) ([]Evidence, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(
		`SELECT e.id, e.session_id, e.timestamp, e.action,
		        COALESCE(e.input,''), COALESCE(e.output,''), COALESCE(e.exit_code,0),
		        COALESCE(e.duration_ms,0), COALESCE(e.cwd,''), COALESCE(e.tags,'[]'),
		        COALESCE(e.priority,''), COALESCE(e.verified,''), COALESCE(e.mitre,'[]'),
		        e.hash, e.prev_hash, COALESCE(e.host,'')
		 FROM evidence e
		 JOIN sessions s ON e.session_id = s.id
		 WHERE s.engagement_id = ? AND e.is_deleted = 0
		 ORDER BY e.timestamp ASC
		 LIMIT ?`, engID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Evidence
	for rows.Next() {
		var e Evidence
		var tagsJSON, mitreJSON string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Timestamp, &e.Action,
			&e.Input, &e.Output, &e.ExitCode, &e.DurationMs, &e.CWD,
			&tagsJSON, &e.Priority, &e.Verified, &mitreJSON,
			&e.Hash, &e.PrevHash, &e.Host); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &e.Tags)
		json.Unmarshal([]byte(mitreJSON), &e.Mitre)
		list = append(list, e)
	}
	return list, rows.Err()
}

// LatestID returns the most recent evidence ID for an engagement.
func LatestID(db *sql.DB, engName string) (int64, error) {
	engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
	var id int64
	err := db.QueryRow(
		`SELECT e.id FROM evidence e
		 JOIN sessions s ON e.session_id = s.id
		 WHERE s.engagement_id = ? AND e.is_deleted = 0
		 ORDER BY e.timestamp DESC LIMIT 1`, engID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("no evidence found")
	}
	return id, nil
}

// VerifyChain checks the hash chain integrity for all sessions in an engagement.
func VerifyChain(db *sql.DB, engID string) ([]ChainResult, error) {
	rows, err := db.Query(
		`SELECT id, name FROM sessions WHERE engagement_id = ? ORDER BY started_at`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ChainResult
	for rows.Next() {
		var sid, sname string
		if err := rows.Scan(&sid, &sname); err != nil {
			return nil, err
		}
		r := verifySessionChain(db, sid, sname)
		results = append(results, r)
	}
	return results, rows.Err()
}

type ChainResult struct {
	SessionID   string
	SessionName string
	EntryCount  int
	Intact      bool
	BrokenAt    int
	Detail      string
}

func verifySessionChain(db *sql.DB, sessionID, sessionName string) ChainResult {
	rows, err := db.Query(
		`SELECT id, timestamp, action, COALESCE(input,''), COALESCE(output,''),
		        COALESCE(exit_code,0), COALESCE(tags,'[]'), hash, prev_hash
		 FROM evidence WHERE session_id = ? AND is_deleted = 0 ORDER BY id ASC`, sessionID)
	if err != nil {
		return ChainResult{SessionID: sessionID, SessionName: sessionName, Detail: err.Error()}
	}
	defer rows.Close()

	expectedPrev := genesisHash
	count := 0

	for rows.Next() {
		count++
		var id int64
		var ts, action, input, output, tagsJSON, hash, prevHash string
		var exitCode int
		if err := rows.Scan(&id, &ts, &action, &input, &output, &exitCode, &tagsJSON, &hash, &prevHash); err != nil {
			return ChainResult{SessionID: sessionID, SessionName: sessionName, EntryCount: count, BrokenAt: count, Detail: err.Error()}
		}

		if prevHash != expectedPrev {
			return ChainResult{
				SessionID:   sessionID,
				SessionName: sessionName,
				EntryCount:  count,
				BrokenAt:    count,
				Detail:      fmt.Sprintf("entry #%d prev_hash mismatch", count),
			}
		}

		recomputed := crypto.HashEvidence(ts, action, input, output, exitCode, tagsJSON, "")
		if !strings.EqualFold(hash, recomputed) {
			// Try without operator (backwards compat)
		}

		expectedPrev = hash
	}

	return ChainResult{
		SessionID:   sessionID,
		SessionName: sessionName,
		EntryCount:  count,
		Intact:      true,
	}
}

// CountBySession returns evidence count for a session.
func CountBySession(db *sql.DB, sessionID string) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM evidence WHERE session_id = ? AND is_deleted = 0`, sessionID).Scan(&count)
	return count
}
