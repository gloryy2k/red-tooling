package engagement

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/rt/internal/audit"
)

type Engagement struct {
	ID        string
	Name      string
	Client    string
	Status    string
	CreatedAt string
	StartDate string
	EndDate   string
}

// Create inserts a new engagement record.
func Create(db *sql.DB, id, name, client string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO engagements (id, name, client, status, start_date, created_at) VALUES (?, ?, ?, 'active', ?, ?)`,
		id, name, client, now, now,
	)
	if err != nil {
		return fmt.Errorf("create engagement: %w", err)
	}
	audit.Log(db, "system", "engagement.create", "engagement", id, map[string]string{"name": name})
	return nil
}

// Get returns a single engagement by ID.
func Get(db *sql.DB, id string) (*Engagement, error) {
	row := db.QueryRow(
		`SELECT id, name, COALESCE(client,''), status, created_at, COALESCE(start_date,''), COALESCE(end_date,'')
		 FROM engagements WHERE id = ?`, id)
	var e Engagement
	if err := row.Scan(&e.ID, &e.Name, &e.Client, &e.Status, &e.CreatedAt, &e.StartDate, &e.EndDate); err != nil {
		return nil, err
	}
	return &e, nil
}

// List returns all engagements.
func List(db *sql.DB) ([]Engagement, error) {
	rows, err := db.Query(
		`SELECT id, name, COALESCE(client,''), status, created_at, COALESCE(start_date,''), COALESCE(end_date,'')
		 FROM engagements ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Engagement
	for rows.Next() {
		var e Engagement
		if err := rows.Scan(&e.ID, &e.Name, &e.Client, &e.Status, &e.CreatedAt, &e.StartDate, &e.EndDate); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
