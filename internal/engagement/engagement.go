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

// Update modifies engagement metadata fields.
func Update(db *sql.DB, id, name, client, startDate, endDate, status, operator string) error {
	_, err := db.Exec(
		`UPDATE engagements SET name = ?, client = ?, start_date = ?, end_date = ?, status = ? WHERE id = ?`,
		name, client, startDate, endDate, status, id,
	)
	if err != nil {
		return fmt.Errorf("update engagement: %w", err)
	}
	audit.Log(db, operator, "engagement.update", "engagement", id,
		map[string]string{"name": name, "client": client, "status": status})
	return nil
}

// GetROE returns the rules of engagement text.
func GetROE(db *sql.DB, id string) string {
	var roe sql.NullString
	db.QueryRow(`SELECT roe FROM engagements WHERE id = ?`, id).Scan(&roe)
	if roe.Valid {
		return roe.String
	}
	return ""
}

// SetROE updates the rules of engagement text.
func SetROE(db *sql.DB, id, roe, operator string) error {
	_, err := db.Exec(`UPDATE engagements SET roe = ? WHERE id = ?`, roe, id)
	if err != nil {
		return fmt.Errorf("set ROE: %w", err)
	}
	audit.Log(db, operator, "engagement.set_roe", "engagement", id, nil)
	return nil
}
