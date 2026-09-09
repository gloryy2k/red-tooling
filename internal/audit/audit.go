package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Entry struct {
	ID         int64
	Timestamp  string
	Operator   string
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	IPAddress  string
}

// Log writes an entry to the audit log.
func Log(db *sql.DB, operator, action, targetType, targetID string, detail interface{}) error {
	var detailJSON string
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			detailJSON = fmt.Sprintf(`{"error":"%s"}`, err.Error())
		} else {
			detailJSON = string(b)
		}
	}

	_, err := db.Exec(
		`INSERT INTO audit_log (timestamp, operator, action, target_type, target_id, detail)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339),
		operator, action, targetType, targetID, detailJSON,
	)
	return err
}

// List returns recent audit log entries.
func List(db *sql.DB, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(
		`SELECT id, timestamp, operator, action, COALESCE(target_type,''), COALESCE(target_id,''), COALESCE(detail,''), COALESCE(ip_address,'')
		 FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Operator, &e.Action, &e.TargetType, &e.TargetID, &e.Detail, &e.IPAddress); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
