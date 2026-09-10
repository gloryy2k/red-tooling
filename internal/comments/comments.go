package comments

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/rt/internal/audit"
)

type Comment struct {
	ID        int64  `json:"id"`
	FindingID int64  `json:"finding_id"`
	Operator  string `json:"operator"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func EnsureTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS finding_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		finding_id INTEGER NOT NULL,
		operator TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_comments_finding ON finding_comments(finding_id)`)
}

func Add(db *sql.DB, findingID int64, operator, content string) (*Comment, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO finding_comments (finding_id, operator, content, created_at) VALUES (?, ?, ?, ?)`,
		findingID, operator, content, now,
	)
	if err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}
	id, _ := res.LastInsertId()
	audit.Log(db, operator, "comment.create", "finding", fmt.Sprintf("%d", findingID), map[string]string{"comment_id": fmt.Sprintf("%d", id)})
	return &Comment{ID: id, FindingID: findingID, Operator: operator, Content: content, CreatedAt: now}, nil
}

func ListByFinding(db *sql.DB, findingID int64) ([]Comment, error) {
	rows, err := db.Query(
		`SELECT id, finding_id, operator, content, created_at
		 FROM finding_comments WHERE finding_id = ? ORDER BY created_at ASC`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.FindingID, &c.Operator, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func Delete(db *sql.DB, commentID int64, operator string) error {
	_, err := db.Exec(`DELETE FROM finding_comments WHERE id = ?`, commentID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	audit.Log(db, operator, "comment.delete", "comment", fmt.Sprintf("%d", commentID), nil)
	return nil
}
