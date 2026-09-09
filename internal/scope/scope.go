package scope

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/user/rt/internal/audit"
)

type Host struct {
	ID        int64  `json:"id"`
	EngID     string `json:"engagement_id"`
	Host      string `json:"Host"`
	Tested    bool   `json:"Tested"`
	TestedAt  string `json:"TestedAt"`
	SessionID string `json:"SessionID"`
}

// Add parses a comma-separated list of hosts/CIDRs and inserts them.
func Add(db *sql.DB, engID, hostList, operator string) (int, error) {
	hosts := parseHosts(hostList)
	count := 0
	for _, h := range hosts {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO scope_hosts (engagement_id, host) VALUES (?, ?)`,
			engID, h,
		)
		if err == nil {
			count++
		}
	}

	audit.Log(db, operator, "scope.add", "scope", engID,
		map[string]interface{}{"hosts": len(hosts), "raw": hostList})

	return count, nil
}

// MarkTested marks a host as tested.
func MarkTested(db *sql.DB, engID, host, sessionID, operator string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`UPDATE scope_hosts SET tested = 1, tested_at = ?, session_id = ?
		 WHERE engagement_id = ? AND host = ?`,
		now, sessionID, engID, host,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("host %q not in scope", host)
	}

	audit.Log(db, operator, "scope.tested", "scope", host, nil)
	return nil
}

// List returns all scope hosts.
func List(db *sql.DB, engID string) ([]Host, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, host, tested, COALESCE(tested_at,''), COALESCE(session_id,'')
		 FROM scope_hosts WHERE engagement_id = ? ORDER BY host`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Host
	for rows.Next() {
		var h Host
		var tested int
		if err := rows.Scan(&h.ID, &h.EngID, &h.Host, &tested, &h.TestedAt, &h.SessionID); err != nil {
			return nil, err
		}
		h.Tested = tested == 1
		list = append(list, h)
	}
	return list, rows.Err()
}

// Untested returns only untested hosts.
func Untested(db *sql.DB, engID string) ([]Host, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, host, tested, COALESCE(tested_at,''), COALESCE(session_id,'')
		 FROM scope_hosts WHERE engagement_id = ? AND tested = 0 ORDER BY host`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Host
	for rows.Next() {
		var h Host
		var tested int
		if err := rows.Scan(&h.ID, &h.EngID, &h.Host, &tested, &h.TestedAt, &h.SessionID); err != nil {
			return nil, err
		}
		h.Tested = tested == 1
		list = append(list, h)
	}
	return list, rows.Err()
}

// Stats returns total and tested counts.
func Stats(db *sql.DB, engID string) (total, tested int) {
	db.QueryRow(`SELECT COUNT(*) FROM scope_hosts WHERE engagement_id = ?`, engID).Scan(&total)
	db.QueryRow(`SELECT COUNT(*) FROM scope_hosts WHERE engagement_id = ? AND tested = 1`, engID).Scan(&tested)
	return
}

func parseHosts(input string) []string {
	var hosts []string
	for _, part := range strings.Split(input, ",") {
		h := strings.TrimSpace(part)
		if h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}
