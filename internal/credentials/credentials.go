package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/user/rt/internal/audit"
)

type Credential struct {
	ID               int64  `json:"id"`
	EngagementID     string `json:"engagement_id"`
	Username         string `json:"Username"`
	CredType         string `json:"CredType"`
	Host             string `json:"Host"`
	SourceEvidenceID int64  `json:"source_evidence_id"`
	FoundAt          string `json:"FoundAt"`
	Notes            string `json:"Notes"`
}

// Store adds a new credential with the secret AES-GCM encrypted (double-layer inside SQLCipher).
func Store(db *sql.DB, engID, username, secret, secretType, host, operator string, evidenceID int64) error {
	encrypted, err := encryptSecret(secret, engID)
	if err != nil {
		return fmt.Errorf("encrypt credential: %w", err)
	}

	var evID interface{}
	if evidenceID > 0 {
		evID = evidenceID
	}

	_, err = db.Exec(
		`INSERT INTO credentials (engagement_id, username, secret_encrypted, secret_type, host, source_evidence_id, notes)
		 VALUES (?, ?, ?, ?, ?, ?, '')`,
		engID, username, encrypted, secretType, host, evID,
	)
	if err != nil {
		return fmt.Errorf("store credential: %w", err)
	}

	audit.Log(db, operator, "cred.create", "credential", username+"@"+host, map[string]string{
		"secret_type": secretType,
	})

	return nil
}

// List returns all credentials for an engagement (secrets NOT decrypted).
func List(db *sql.DB, engID string) ([]Credential, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, COALESCE(username,''), COALESCE(secret_type,'password'),
		        COALESCE(host,''), COALESCE(source_evidence_id,0), created_at, COALESCE(notes,'')
		 FROM credentials WHERE engagement_id = ? ORDER BY created_at DESC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Credential
	for rows.Next() {
		var c Credential
		if err := rows.Scan(&c.ID, &c.EngagementID, &c.Username, &c.CredType,
			&c.Host, &c.SourceEvidenceID, &c.FoundAt, &c.Notes); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Reveal decrypts and returns the secret for a credential. Audit-logged.
func Reveal(db *sql.DB, engID string, credID int64, operator string) (string, error) {
	var encrypted []byte
	var username, host string
	err := db.QueryRow(
		`SELECT secret_encrypted, COALESCE(username,''), COALESCE(host,'')
		 FROM credentials WHERE id = ? AND engagement_id = ?`, credID, engID,
	).Scan(&encrypted, &username, &host)
	if err != nil {
		return "", fmt.Errorf("credential not found: %w", err)
	}

	secret, err := decryptSecret(encrypted, engID)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}

	audit.Log(db, operator, "cred.view", "credential", fmt.Sprintf("#%d %s@%s", credID, username, host), nil)

	return secret, nil
}

// RevealAll returns all credentials with secrets decrypted. Audit-logged.
func RevealAll(db *sql.DB, engID, operator string) ([]CredentialWithSecret, error) {
	rows, err := db.Query(
		`SELECT id, COALESCE(username,''), secret_encrypted, COALESCE(secret_type,'password'),
		        COALESCE(host,''), created_at
		 FROM credentials WHERE engagement_id = ? ORDER BY created_at DESC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []CredentialWithSecret
	for rows.Next() {
		var c CredentialWithSecret
		var encrypted []byte
		if err := rows.Scan(&c.ID, &c.Username, &encrypted, &c.SecretType, &c.Host, &c.FoundAt); err != nil {
			return nil, err
		}
		secret, err := decryptSecret(encrypted, engID)
		if err != nil {
			c.Secret = "[decrypt error]"
		} else {
			c.Secret = secret
		}
		list = append(list, c)
	}

	audit.Log(db, operator, "cred.view", "credential", "all", map[string]string{"count": fmt.Sprintf("%d", len(list))})
	return list, rows.Err()
}

type CredentialWithSecret struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	SecretType string `json:"secret_type"`
	Host      string `json:"host"`
	FoundAt   string `json:"found_at"`
}

// ExportCSV returns credentials as CSV. Audit-logged.
func ExportCSV(db *sql.DB, engID, operator string) (string, error) {
	creds, err := RevealAll(db, engID, operator)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("username,secret,type,host,created_at\n")
	for _, c := range creds {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			csvEscape(c.Username), csvEscape(c.Secret), c.SecretType, csvEscape(c.Host), c.FoundAt))
	}

	audit.Log(db, operator, "cred.export", "credential", "csv", map[string]string{"count": fmt.Sprintf("%d", len(creds))})
	return sb.String(), nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// Encryption: AES-256-GCM using SHA-256(engagementID) as key.
// This is a second layer inside the already-encrypted SQLCipher database.
func encryptSecret(plaintext, engID string) ([]byte, error) {
	key := deriveCredKey(engID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func decryptSecret(ciphertext []byte, engID string) (string, error) {
	key := deriveCredKey(engID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func deriveCredKey(engID string) []byte {
	h := sha256.Sum256([]byte("rt-cred-key:" + engID))
	return h[:]
}

// Count returns total credentials for an engagement.
func Count(db *sql.DB, engID string) int {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM credentials WHERE engagement_id = ?`, engID).Scan(&n)
	return n
}
