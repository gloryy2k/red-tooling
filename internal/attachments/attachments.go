package attachments

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/user/rt/internal/audit"
)

type Attachment struct {
	ID          int64
	EvidenceID  int64
	Filename    string
	Caption     string
	Filetype    string
	SizeBytes   int64
	ContentHash string
	CreatedAt   string
}

var allowedTypes = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true,
	"pdf": true, "txt": true, "xml": true, "json": true,
	"pcap": true, "pfx": true, "pem": true, "csv": true,
	"html": true, "log": true, "conf": true, "cfg": true,
}

var safeFilename = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// Store reads a file from disk and stores it as a BLOB in the database.
func Store(db *sql.DB, evidenceID int64, filePath, caption, operator string) (*Attachment, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	filename := sanitizeFilename(filepath.Base(filePath))
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if !allowedTypes[strings.ToLower(ext)] {
		return nil, fmt.Errorf("file type %q not allowed. Allowed: %s", ext, allowedTypesList())
	}

	if len(data) > 50*1024*1024 {
		return nil, fmt.Errorf("file too large (%d bytes). Maximum: 50MB", len(data))
	}

	h := sha256.Sum256(data)
	contentHash := hex.EncodeToString(h[:])

	filetype := detectFiletype(ext, data)

	res, err := db.Exec(
		`INSERT INTO attachments (evidence_id, filename, caption, filetype, size_bytes, content, content_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		evidenceID, filename, caption, filetype, len(data), data, contentHash,
	)
	if err != nil {
		return nil, fmt.Errorf("store attachment: %w", err)
	}

	id, _ := res.LastInsertId()
	audit.Log(db, operator, "attachment.create", "attachment", fmt.Sprintf("%d", id),
		map[string]string{"filename": filename, "evidence_id": fmt.Sprintf("%d", evidenceID)})

	return &Attachment{
		ID:          id,
		EvidenceID:  evidenceID,
		Filename:    filename,
		Caption:     caption,
		Filetype:    filetype,
		SizeBytes:   int64(len(data)),
		ContentHash: contentHash,
	}, nil
}

// List returns attachments for an evidence entry.
func ListByEvidence(db *sql.DB, evidenceID int64) ([]Attachment, error) {
	rows, err := db.Query(
		`SELECT id, evidence_id, filename, COALESCE(caption,''), COALESCE(filetype,''),
		        size_bytes, content_hash, created_at
		 FROM attachments WHERE evidence_id = ? ORDER BY created_at`, evidenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.EvidenceID, &a.Filename, &a.Caption,
			&a.Filetype, &a.SizeBytes, &a.ContentHash, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ListByEngagement returns all attachments for an engagement.
func ListByEngagement(db *sql.DB, engID string) ([]Attachment, error) {
	rows, err := db.Query(
		`SELECT a.id, a.evidence_id, a.filename, COALESCE(a.caption,''), COALESCE(a.filetype,''),
		        a.size_bytes, a.content_hash, a.created_at
		 FROM attachments a
		 JOIN evidence e ON a.evidence_id = e.id
		 JOIN sessions s ON e.session_id = s.id
		 WHERE s.engagement_id = ?
		 ORDER BY a.created_at DESC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.EvidenceID, &a.Filename, &a.Caption,
			&a.Filetype, &a.SizeBytes, &a.ContentHash, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// Export writes an attachment to a file on disk.
func Export(db *sql.DB, attachmentID int64, outDir, operator string) (string, error) {
	var filename string
	var content []byte
	err := db.QueryRow(
		`SELECT filename, content FROM attachments WHERE id = ?`, attachmentID,
	).Scan(&filename, &content)
	if err != nil {
		return "", fmt.Errorf("attachment not found: %w", err)
	}

	outPath := filepath.Join(outDir, filename)
	if err := os.WriteFile(outPath, content, 0600); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	audit.Log(db, operator, "attachment.export", "attachment", fmt.Sprintf("%d", attachmentID),
		map[string]string{"filename": filename})

	return outPath, nil
}

// GetContent returns an attachment's raw content and metadata for serving.
func GetContent(db *sql.DB, attachmentID int64) ([]byte, string, string, error) {
	var content []byte
	var filename, filetype string
	err := db.QueryRow(
		`SELECT content, filename, COALESCE(filetype,'') FROM attachments WHERE id = ?`, attachmentID,
	).Scan(&content, &filename, &filetype)
	if err != nil {
		return nil, "", "", fmt.Errorf("attachment not found: %w", err)
	}
	return content, filename, filetype, nil
}

// ListByFinding returns attachments for evidence entries linked to a finding.
func ListByFinding(db *sql.DB, findingID int64) ([]Attachment, error) {
	rows, err := db.Query(
		`SELECT a.id, a.evidence_id, a.filename, COALESCE(a.caption,''), COALESCE(a.filetype,''),
		        a.size_bytes, a.content_hash, a.created_at
		 FROM attachments a
		 WHERE a.evidence_id IN (
		   SELECT json_each.value FROM findings f, json_each(f.evidence_ids) WHERE f.id = ?
		 )
		 ORDER BY a.created_at DESC`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.EvidenceID, &a.Filename, &a.Caption,
			&a.Filetype, &a.SizeBytes, &a.ContentHash, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// Count returns total attachments in the database.
func Count(db *sql.DB) int {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM attachments`).Scan(&n)
	return n
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = safeFilename.ReplaceAllString(name, "_")
	if len(name) > 255 {
		name = name[:255]
	}
	if name == "" || name == "." {
		name = "attachment"
	}
	return name
}

func detectFiletype(ext string, data []byte) string {
	if len(data) >= 4 {
		// PNG magic
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "screenshot"
		}
		// JPEG magic
		if data[0] == 0xFF && data[1] == 0xD8 {
			return "screenshot"
		}
		// PDF magic
		if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
			return "pdf"
		}
		// GIF magic
		if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
			return "screenshot"
		}
	}

	switch strings.ToLower(ext) {
	case "pcap", "pcapng":
		return "pcap"
	case "pfx", "p12":
		return "cert"
	case "pem", "crt", "cer":
		return "cert"
	case "xml":
		return "config"
	case "json", "csv", "txt", "log":
		return ext
	case "conf", "cfg", "ini":
		return "config"
	}
	return ext
}

func allowedTypesList() string {
	var types []string
	for t := range allowedTypes {
		types = append(types, t)
	}
	return strings.Join(types, ", ")
}
