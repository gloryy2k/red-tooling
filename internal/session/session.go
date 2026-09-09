package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/config"
)

type Session struct {
	ID           string
	EngagementID string
	Name         string
	Source       string
	Operator     string
	Status       string
	StartedAt    string
	EndedAt      string
}

// Create inserts a new session.
func Create(db *sql.DB, id, engID, name, source, operator string) error {
	var opVal interface{}
	if operator != "" {
		opVal = operator
	}
	_, err := db.Exec(
		`INSERT INTO sessions (id, engagement_id, name, source, operator, status, started_at)
		 VALUES (?, ?, ?, ?, ?, 'active', ?)`,
		id, engID, name, source, opVal,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	audit.Log(db, operator, "session.start", "session", id, map[string]string{"name": name, "source": source})
	return nil
}

// Stop marks a session as stopped.
func Stop(db *sql.DB, id, operator string) error {
	_, err := db.Exec(
		`UPDATE sessions SET status = 'stopped', ended_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("stop session: %w", err)
	}
	audit.Log(db, operator, "session.stop", "session", id, nil)
	return nil
}

// GetActive returns the currently active session for an engagement, if any.
func GetActive(db *sql.DB, engID string) (*Session, error) {
	row := db.QueryRow(
		`SELECT id, engagement_id, name, source, COALESCE(operator,''), status, started_at, COALESCE(ended_at,'')
		 FROM sessions WHERE engagement_id = ? AND status = 'active' LIMIT 1`, engID)
	var s Session
	if err := row.Scan(&s.ID, &s.EngagementID, &s.Name, &s.Source, &s.Operator, &s.Status, &s.StartedAt, &s.EndedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// List returns all sessions for an engagement.
func List(db *sql.DB, engID string) ([]Session, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, name, source, COALESCE(operator,''), status, started_at, COALESCE(ended_at,'')
		 FROM sessions WHERE engagement_id = ? ORDER BY started_at DESC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.EngagementID, &s.Name, &s.Source, &s.Operator, &s.Status, &s.StartedAt, &s.EndedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ActiveSessionFile stores the currently active session ID for quick lookup.
func ActiveSessionFile() string {
	return filepath.Join(config.Home(), "active_session")
}

func SetActiveSession(sessionID string) error {
	return os.WriteFile(ActiveSessionFile(), []byte(sessionID), 0600)
}

func GetActiveSessionID() string {
	data, err := os.ReadFile(ActiveSessionFile())
	if err != nil {
		return ""
	}
	return string(data)
}

func ClearActiveSession() {
	os.Remove(ActiveSessionFile())
}
