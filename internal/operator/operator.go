package operator

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/user/rt/internal/audit"
)

type Operator struct {
	ID         string
	EngID      string
	Role       string
	APIKey     string // plaintext, only returned on creation
	APIKeyHash string
	CreatedAt  string
	LastSeenAt string
}

var validRoles = map[string]bool{
	"lead": true, "operator": true, "reviewer": true, "viewer": true,
}

// RolePermissions defines what each role can access.
var RolePermissions = map[string]map[string]bool{
	"lead": {
		"overview": true, "live": true, "findings": true, "timeline": true,
		"creds": true, "audit": true, "sessions": true, "report": true,
		"operators": true, "evidence": true, "verify": true, "export": true,
	},
	"operator": {
		"overview": true, "live": true, "findings": true, "timeline": true,
		"creds": true, "sessions": true, "evidence": true, "report": true,
		"export": true,
	},
	"reviewer": {
		"overview": true, "findings": true, "timeline": true,
		"sessions": true, "evidence": true, "report": true, "verify": true,
		"export": true,
	},
	"viewer": {
		"overview": true, "findings": true, "timeline": true,
		"sessions": true, "report": true,
	},
}

func HasPermission(role, resource string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	return perms[resource]
}

func generateAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rt_key_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// Add creates a new operator with an auto-generated API key.
func Add(db *sql.DB, engID, operatorID, role, creator string) (*Operator, error) {
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role %q. Valid: lead, operator, reviewer, viewer", role)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate API key: %w", err)
	}

	keyHash := hashAPIKey(apiKey)
	_, err = db.Exec(
		`INSERT INTO operators (id, engagement_id, role, api_key_hash) VALUES (?, ?, ?, ?)`,
		operatorID, engID, role, keyHash,
	)
	if err != nil {
		return nil, fmt.Errorf("add operator: %w", err)
	}

	audit.Log(db, creator, "operator.add", "operator", operatorID,
		map[string]string{"role": role})

	return &Operator{
		ID:         operatorID,
		EngID:      engID,
		Role:       role,
		APIKey:     apiKey,
		APIKeyHash: keyHash,
	}, nil
}

// List returns all operators for an engagement.
func List(db *sql.DB, engID string) ([]Operator, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, role, COALESCE(api_key_hash,''), created_at, COALESCE(last_seen_at,'')
		 FROM operators WHERE engagement_id = ? ORDER BY created_at`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Operator
	for rows.Next() {
		var o Operator
		if err := rows.Scan(&o.ID, &o.EngID, &o.Role, &o.APIKeyHash, &o.CreatedAt, &o.LastSeenAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// Authenticate validates an API key and returns the operator.
func Authenticate(db *sql.DB, engID, apiKey string) (*Operator, error) {
	keyHash := hashAPIKey(apiKey)
	var o Operator
	err := db.QueryRow(
		`SELECT id, engagement_id, role, COALESCE(api_key_hash,''), COALESCE(created_at,''), COALESCE(last_seen_at,'')
		 FROM operators WHERE engagement_id = ? AND api_key_hash = ?`, engID, keyHash,
	).Scan(&o.ID, &o.EngID, &o.Role, &o.APIKeyHash, &o.CreatedAt, &o.LastSeenAt)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	db.Exec(`UPDATE operators SET last_seen_at = datetime('now') WHERE id = ? AND engagement_id = ?`, o.ID, engID)
	return &o, nil
}

// RotateKey generates a new API key for an existing operator.
func RotateKey(db *sql.DB, engID, operatorID, creator string) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	keyHash := hashAPIKey(apiKey)
	res, err := db.Exec(
		`UPDATE operators SET api_key_hash = ? WHERE id = ? AND engagement_id = ?`,
		keyHash, operatorID, engID,
	)
	if err != nil {
		return "", err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return "", fmt.Errorf("operator %q not found", operatorID)
	}

	audit.Log(db, creator, "operator.rotate_key", "operator", operatorID, nil)
	return apiKey, nil
}

// Get returns a single operator.
func Get(db *sql.DB, engID, operatorID string) (*Operator, error) {
	var o Operator
	err := db.QueryRow(
		`SELECT id, engagement_id, role, COALESCE(api_key_hash,''), COALESCE(created_at,''), COALESCE(last_seen_at,'')
		 FROM operators WHERE id = ? AND engagement_id = ?`, operatorID, engID,
	).Scan(&o.ID, &o.EngID, &o.Role, &o.APIKeyHash, &o.CreatedAt, &o.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
