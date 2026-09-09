package findings

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/user/rt/internal/audit"
)

type Finding struct {
	ID             int64
	EngagementID   string
	Title          string
	Description    string
	Priority       string
	Verified       string
	Recommendation string
	EvidenceIDs    []int64
	Mitre          []string
	CreatedAt      string
	VerifiedBy     string
	VerifiedAt     string
	Notes          string
}

// Create inserts a new finding.
func Create(db *sql.DB, engID, title, description, priority, operator string, evidenceIDs []int64, mitre []string) (*Finding, error) {
	evJSON, _ := json.Marshal(evidenceIDs)
	mitreJSON, _ := json.Marshal(mitre)

	res, err := db.Exec(
		`INSERT INTO findings (engagement_id, title, description, priority, evidence_ids, mitre)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		engID, title, description, priority, string(evJSON), string(mitreJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("create finding: %w", err)
	}

	id, _ := res.LastInsertId()
	audit.Log(db, operator, "finding.create", "finding", fmt.Sprintf("%d", id), map[string]string{"title": title, "priority": priority})

	return &Finding{
		ID:           id,
		EngagementID: engID,
		Title:        title,
		Description:  description,
		Priority:     priority,
		Verified:     "unverified",
		EvidenceIDs:  evidenceIDs,
		Mitre:        mitre,
	}, nil
}

// List returns findings for an engagement.
func List(db *sql.DB, engID string) ([]Finding, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, title, COALESCE(description,''), COALESCE(priority,''),
		        verified, COALESCE(recommendation,''), COALESCE(evidence_ids,'[]'),
		        COALESCE(mitre,'[]'), created_at, COALESCE(verified_by,''),
		        COALESCE(verified_at,''), COALESCE(notes,'')
		 FROM findings WHERE engagement_id = ? ORDER BY
		   CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END,
		   created_at DESC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Finding
	for rows.Next() {
		var f Finding
		var evJSON, mitreJSON string
		if err := rows.Scan(&f.ID, &f.EngagementID, &f.Title, &f.Description,
			&f.Priority, &f.Verified, &f.Recommendation, &evJSON,
			&mitreJSON, &f.CreatedAt, &f.VerifiedBy, &f.VerifiedAt, &f.Notes); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(evJSON), &f.EvidenceIDs)
		json.Unmarshal([]byte(mitreJSON), &f.Mitre)
		list = append(list, f)
	}
	return list, rows.Err()
}

// Get returns a single finding.
func Get(db *sql.DB, id int64) (*Finding, error) {
	var f Finding
	var evJSON, mitreJSON string
	err := db.QueryRow(
		`SELECT id, engagement_id, title, COALESCE(description,''), COALESCE(priority,''),
		        verified, COALESCE(recommendation,''), COALESCE(evidence_ids,'[]'),
		        COALESCE(mitre,'[]'), created_at, COALESCE(verified_by,''),
		        COALESCE(verified_at,''), COALESCE(notes,'')
		 FROM findings WHERE id = ?`, id,
	).Scan(&f.ID, &f.EngagementID, &f.Title, &f.Description,
		&f.Priority, &f.Verified, &f.Recommendation, &evJSON,
		&mitreJSON, &f.CreatedAt, &f.VerifiedBy, &f.VerifiedAt, &f.Notes)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(evJSON), &f.EvidenceIDs)
	json.Unmarshal([]byte(mitreJSON), &f.Mitre)
	return &f, nil
}

// Verify marks a finding as confirmed or false-positive.
func Verify(db *sql.DB, id int64, status, operator, note string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	updateNote := ""
	if note != "" {
		updateNote = ", notes = COALESCE(notes,'') || '\n' || ?"
	}

	query := fmt.Sprintf(
		`UPDATE findings SET verified = ?, verified_by = ?, verified_at = ?%s WHERE id = ?`,
		updateNote,
	)

	var err error
	if note != "" {
		_, err = db.Exec(query, status, operator, now, note, id)
	} else {
		_, err = db.Exec(query, status, operator, now, id)
	}
	if err != nil {
		return fmt.Errorf("verify finding: %w", err)
	}

	audit.Log(db, operator, "finding.verify", "finding", fmt.Sprintf("%d", id), map[string]string{"status": status})
	return nil
}

// Merge combines multiple findings into one. Keeps the first, deletes the rest.
func Merge(db *sql.DB, ids []int64, newTitle, operator string) (*Finding, error) {
	if len(ids) < 2 {
		return nil, fmt.Errorf("merge requires at least 2 findings")
	}

	// Collect all evidence IDs and highest priority
	evSet := make(map[int64]bool)
	mitreSet := make(map[string]bool)
	bestPriority := ""
	var descriptions []string
	var engID string
	priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0}

	for _, id := range ids {
		f, err := Get(db, id)
		if err != nil {
			return nil, fmt.Errorf("finding #%d not found: %w", id, err)
		}
		if engID == "" {
			engID = f.EngagementID
		}
		for _, eid := range f.EvidenceIDs {
			evSet[eid] = true
		}
		for _, m := range f.Mitre {
			mitreSet[m] = true
		}
		if priorityOrder[f.Priority] > priorityOrder[bestPriority] {
			bestPriority = f.Priority
		}
		if f.Description != "" {
			descriptions = append(descriptions, f.Description)
		}
	}

	var allEvIDs []int64
	for eid := range evSet {
		allEvIDs = append(allEvIDs, eid)
	}
	var allMitre []string
	for m := range mitreSet {
		allMitre = append(allMitre, m)
	}

	mergedDesc := strings.Join(descriptions, "\n---\n")

	// Create merged finding
	merged, err := Create(db, engID, newTitle, mergedDesc, bestPriority, operator, allEvIDs, allMitre)
	if err != nil {
		return nil, err
	}

	// Delete originals
	for _, id := range ids {
		db.Exec(`DELETE FROM findings WHERE id = ?`, id)
	}

	audit.Log(db, operator, "finding.merge", "finding", fmt.Sprintf("%d", merged.ID),
		map[string]interface{}{"merged_from": ids, "title": newTitle})

	return merged, nil
}

// SetRecommendation sets the recommendation for a finding.
func SetRecommendation(db *sql.DB, id int64, recommendation, operator string) error {
	_, err := db.Exec(`UPDATE findings SET recommendation = ? WHERE id = ?`, recommendation, id)
	if err != nil {
		return fmt.Errorf("set recommendation: %w", err)
	}
	audit.Log(db, operator, "finding.update", "finding", fmt.Sprintf("%d", id), map[string]string{"field": "recommendation"})
	return nil
}

// Count returns total findings for an engagement.
func Count(db *sql.DB, engID string) int {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ?`, engID).Scan(&n)
	return n
}

// CountByStatus returns finding counts by verification status.
func CountByStatus(db *sql.DB, engID string) (unverified, confirmed, falsePositive int) {
	db.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ? AND verified = 'unverified'`, engID).Scan(&unverified)
	db.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ? AND verified = 'confirmed'`, engID).Scan(&confirmed)
	db.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ? AND verified = 'false-positive'`, engID).Scan(&falsePositive)
	return
}
