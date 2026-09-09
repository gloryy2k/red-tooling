package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Result struct {
	ID        int64
	Timestamp string
	Action    string
	Input     string
	Output    string
	Tags      []string
	Priority  string
	Session   string
}

// Search performs full-text search across evidence input and output.
func Search(db *sql.DB, engID, query string, tag, priority string, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = 50
	}

	conditions := []string{
		"s.engagement_id = ?",
		"e.is_deleted = 0",
		"(e.input LIKE ? OR e.output LIKE ?)",
	}
	args := []interface{}{engID, "%" + query + "%", "%" + query + "%"}

	if tag != "" {
		conditions = append(conditions, "e.tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}
	if priority != "" {
		conditions = append(conditions, "e.priority = ?")
		args = append(args, priority)
	}

	args = append(args, limit)

	q := fmt.Sprintf(
		`SELECT e.id, e.timestamp, e.action, COALESCE(e.input,''), COALESCE(e.output,''),
		        COALESCE(e.tags,'[]'), COALESCE(e.priority,''), s.name
		 FROM evidence e JOIN sessions s ON e.session_id = s.id
		 WHERE %s
		 ORDER BY e.timestamp DESC LIMIT ?`,
		strings.Join(conditions, " AND "),
	)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var tagsJSON string
		if err := rows.Scan(&r.ID, &r.Timestamp, &r.Action, &r.Input, &r.Output,
			&tagsJSON, &r.Priority, &r.Session); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &r.Tags)
		results = append(results, r)
	}
	return results, rows.Err()
}

// Query executes a read-only SQL query with safety guards.
func Query(db *sql.DB, query string) ([]map[string]interface{}, error) {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if !strings.HasPrefix(upper, "SELECT") {
		return nil, fmt.Errorf("only SELECT queries allowed (read-only)")
	}

	dangerous := []string{" DROP ", " DELETE ", " INSERT ", " UPDATE ", " ALTER ", " CREATE ", " TRUNCATE ",
		";DROP ", ";DELETE ", ";INSERT ", ";UPDATE ", ";ALTER ", ";CREATE ", ";TRUNCATE "}
	padded := " " + upper + " "
	for _, d := range dangerous {
		if strings.Contains(padded, d) {
			return nil, fmt.Errorf("query contains disallowed keyword: %s", strings.TrimSpace(d))
		}
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
