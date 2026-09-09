package checklist

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/rt/internal/audit"
)

type Item struct {
	ID         int64  `json:"id"`
	EngID      string `json:"engagement_id"`
	Category   string `json:"category"`
	Item       string `json:"item"`
	Done       bool   `json:"done"`
	DoneAt     string `json:"done_at"`
	EvidenceID int64  `json:"evidence_id"`
	Notes      string `json:"notes"`
}

// Add inserts a checklist item.
func Add(db *sql.DB, engID, category, item string) error {
	_, err := db.Exec(
		`INSERT INTO checklist (engagement_id, category, item) VALUES (?, ?, ?)`,
		engID, category, item,
	)
	return err
}

// Check marks a checklist item as done.
func Check(db *sql.DB, id int64, operator string, evidenceID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var evID interface{}
	if evidenceID > 0 {
		evID = evidenceID
	}
	_, err := db.Exec(
		`UPDATE checklist SET done = 1, done_at = ?, evidence_id = ? WHERE id = ?`,
		now, evID, id,
	)
	if err != nil {
		return err
	}
	audit.Log(db, operator, "checklist.check", "checklist", fmt.Sprintf("%d", id), nil)
	return nil
}

// Uncheck marks a checklist item as not done.
func Uncheck(db *sql.DB, id int64) error {
	_, err := db.Exec(
		`UPDATE checklist SET done = 0, done_at = NULL, evidence_id = NULL WHERE id = ?`, id)
	return err
}

// List returns all checklist items for an engagement.
func List(db *sql.DB, engID string) ([]Item, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, COALESCE(category,''), item, done,
		        COALESCE(done_at,''), COALESCE(evidence_id,0), COALESCE(notes,'')
		 FROM checklist WHERE engagement_id = ?
		 ORDER BY category, id`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Item
	for rows.Next() {
		var it Item
		var done int
		if err := rows.Scan(&it.ID, &it.EngID, &it.Category, &it.Item,
			&done, &it.DoneAt, &it.EvidenceID, &it.Notes); err != nil {
			return nil, err
		}
		it.Done = done == 1
		list = append(list, it)
	}
	return list, rows.Err()
}

// Stats returns done/total counts.
func Stats(db *sql.DB, engID string) (done, total int) {
	db.QueryRow(`SELECT COUNT(*) FROM checklist WHERE engagement_id = ?`, engID).Scan(&total)
	db.QueryRow(`SELECT COUNT(*) FROM checklist WHERE engagement_id = ? AND done = 1`, engID).Scan(&done)
	return
}

// LoadPreset loads a named checklist preset.
func LoadPreset(db *sql.DB, engID, preset string) error {
	switch preset {
	case "ptes":
		_, err := LoadPTES(db, engID)
		return err
	default:
		return fmt.Errorf("unknown preset: %s", preset)
	}
}

// LoadPTES loads the standard PTES checklist template.
func LoadPTES(db *sql.DB, engID string) (int, error) {
	items := []struct{ cat, item string }{
		{"Pre-engagement", "Scope definition"},
		{"Pre-engagement", "Rules of engagement signed"},
		{"Pre-engagement", "Emergency contacts exchanged"},
		{"Pre-engagement", "Timeline agreed"},
		{"Intelligence Gathering", "OSINT / passive recon"},
		{"Intelligence Gathering", "DNS enumeration"},
		{"Intelligence Gathering", "Network scanning"},
		{"Intelligence Gathering", "Service enumeration"},
		{"Intelligence Gathering", "Web application mapping"},
		{"Vulnerability Analysis", "Automated vulnerability scanning"},
		{"Vulnerability Analysis", "Manual vulnerability verification"},
		{"Vulnerability Analysis", "Web application testing"},
		{"Vulnerability Analysis", "Network vulnerability assessment"},
		{"Exploitation", "Initial access attempts"},
		{"Exploitation", "Credential attacks"},
		{"Exploitation", "Web exploitation"},
		{"Exploitation", "Network exploitation"},
		{"Post-Exploitation", "Privilege escalation"},
		{"Post-Exploitation", "Lateral movement"},
		{"Post-Exploitation", "Data exfiltration (simulated)"},
		{"Post-Exploitation", "Persistence mechanisms"},
		{"Post-Exploitation", "Domain dominance"},
		{"Reporting", "Evidence review"},
		{"Reporting", "Finding documentation"},
		{"Reporting", "Recommendation writing"},
		{"Reporting", "Report generation"},
		{"Reporting", "Client debrief"},
	}

	count := 0
	for _, it := range items {
		if err := Add(db, engID, it.cat, it.item); err == nil {
			count++
		}
	}
	return count, nil
}
