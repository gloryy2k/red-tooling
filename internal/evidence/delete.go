package evidence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/user/rt/internal/audit"
)

// SoftDelete marks evidence entries as deleted without removing them.
func SoftDelete(db *sql.DB, ids []int64, operator string) error {
	if len(ids) == 0 {
		return fmt.Errorf("no evidence IDs provided")
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`UPDATE evidence SET is_deleted = 1 WHERE id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	res, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}

	affected, _ := res.RowsAffected()
	audit.Log(db, operator, "evidence.delete", "evidence", fmt.Sprintf("%v", ids),
		map[string]interface{}{"count": affected, "ids": ids})

	return nil
}

// Redact replaces the output of an evidence entry with a redaction marker.
func Redact(db *sql.DB, id int64, reason, operator string) error {
	marker := fmt.Sprintf("[REDACTED: %s]", reason)

	_, err := db.Exec(
		`UPDATE evidence SET output = ?, input = CASE WHEN action = 'command' THEN input ELSE ? END WHERE id = ? AND is_deleted = 0`,
		marker, marker, id,
	)
	if err != nil {
		return fmt.Errorf("redact evidence: %w", err)
	}

	audit.Log(db, operator, "evidence.redact", "evidence", fmt.Sprintf("%d", id),
		map[string]string{"reason": reason})

	return nil
}

// Get returns a single evidence entry by ID.
func Get(db *sql.DB, id int64) (*Evidence, error) {
	var e Evidence
	var tagsJSON, mitreJSON string
	err := db.QueryRow(
		`SELECT id, session_id, timestamp, action,
		        COALESCE(input,''), COALESCE(output,''), COALESCE(exit_code,0),
		        COALESCE(duration_ms,0), COALESCE(cwd,''), COALESCE(tags,'[]'),
		        COALESCE(priority,''), COALESCE(verified,''), COALESCE(mitre,'[]'),
		        hash, prev_hash
		 FROM evidence WHERE id = ? AND is_deleted = 0`, id,
	).Scan(&e.ID, &e.SessionID, &e.Timestamp, &e.Action,
		&e.Input, &e.Output, &e.ExitCode, &e.DurationMs, &e.CWD,
		&tagsJSON, &e.Priority, &e.Verified, &mitreJSON,
		&e.Hash, &e.PrevHash)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
