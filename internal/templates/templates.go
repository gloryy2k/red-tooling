package templates

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/user/rt/internal/audit"
)

type Template struct {
	ID           int64  `json:"id"`
	EngagementID string `json:"engagement_id"`
	Name         string `json:"name"`
	Format       string `json:"format"`
	Content      string `json:"content"`
	IsDefault    bool   `json:"is_default"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func EnsureTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS report_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		engagement_id TEXT NOT NULL,
		name TEXT NOT NULL,
		format TEXT NOT NULL DEFAULT 'markdown',
		content TEXT NOT NULL DEFAULT '',
		is_default INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
}

func Create(db *sql.DB, engID, name, format, content string, isDefault bool, operator string) (*Template, error) {
	defInt := 0
	if isDefault {
		defInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO report_templates (engagement_id, name, format, content, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		engID, name, format, content, defInt, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	id, _ := res.LastInsertId()
	audit.Log(db, operator, "template.create", "template", fmt.Sprintf("%d", id), map[string]string{"name": name})
	return &Template{ID: id, EngagementID: engID, Name: name, Format: format, Content: content, IsDefault: isDefault, CreatedAt: now, UpdatedAt: now}, nil
}

func List(db *sql.DB, engID string) ([]Template, error) {
	rows, err := db.Query(
		`SELECT id, engagement_id, name, format, content, is_default, created_at, updated_at
		 FROM report_templates WHERE engagement_id = ? ORDER BY is_default DESC, name ASC`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Template
	for rows.Next() {
		var t Template
		var isDef int
		if err := rows.Scan(&t.ID, &t.EngagementID, &t.Name, &t.Format, &t.Content, &isDef, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.IsDefault = isDef == 1
		list = append(list, t)
	}
	return list, rows.Err()
}

func Get(db *sql.DB, id int64) (*Template, error) {
	var t Template
	var isDef int
	err := db.QueryRow(
		`SELECT id, engagement_id, name, format, content, is_default, created_at, updated_at
		 FROM report_templates WHERE id = ?`, id,
	).Scan(&t.ID, &t.EngagementID, &t.Name, &t.Format, &t.Content, &isDef, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	t.IsDefault = isDef == 1
	return &t, nil
}

func Update(db *sql.DB, id int64, name, content string, operator string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE report_templates SET name = ?, content = ?, updated_at = ? WHERE id = ?`,
		name, content, now, id,
	)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}
	audit.Log(db, operator, "template.update", "template", fmt.Sprintf("%d", id), map[string]string{"name": name})
	return nil
}

func Delete(db *sql.DB, id int64, operator string) error {
	_, err := db.Exec(`DELETE FROM report_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	audit.Log(db, operator, "template.delete", "template", fmt.Sprintf("%d", id), nil)
	return nil
}

func SeedDefaults(db *sql.DB, engID, operator string) error {
	existing, _ := List(db, engID)
	if len(existing) > 0 {
		return nil
	}

	defaults := []struct {
		Name    string
		Content string
	}{
		{"Executive Summary", execTemplate},
		{"Technical Report", techTemplate},
		{"Finding Card", findingCardTemplate},
	}

	for _, d := range defaults {
		_, err := Create(db, engID, d.Name, "markdown", d.Content, false, operator)
		if err != nil {
			return err
		}
	}
	return nil
}

const execTemplate = `# Executive Summary — {{.Engagement.Name}}

**Client:** {{.Engagement.Client}}
**Date:** {{.GeneratedAt}}
**Classification:** CONFIDENTIAL

## Overview

This report summarizes the findings from the penetration test conducted against {{.Engagement.Name}}.

## Key Findings

| # | Finding | Severity | Status |
|---|---------|----------|--------|
{{range .Findings}}- | {{.ID}} | {{.Title}} | {{.Priority}} | {{.Verified}} |
{{end}}

## Risk Summary

- **Critical:** {{.Stats.Critical}} findings
- **High:** {{.Stats.High}} findings
- **Medium:** {{.Stats.Medium}} findings
- **Low:** {{.Stats.Low}} findings

## Recommendations

{{range .Findings}}{{if .Recommendation}}### {{.Title}}
{{.Recommendation}}

{{end}}{{end}}
`

const techTemplate = `# Technical Report — {{.Engagement.Name}}

**Client:** {{.Engagement.Client}}
**Date:** {{.GeneratedAt}}
**Operator(s):** {{range .Sessions}}{{.Operator}} {{end}}

## Scope

{{range .Scope}}
- {{.Host}} ({{if .TestedAt}}tested{{else}}untested{{end}})
{{end}}

## Findings

{{range .Findings}}
### [{{.Priority | upper}}] {{.Title}}

**ID:** {{.ID}} | **MITRE:** {{range .Mitre}}{{.}} {{end}} | **Status:** {{.Verified}}

{{.Description}}

**Evidence:** {{range .EvidenceIDs}}#{{.}} {{end}}

{{if .Recommendation}}**Recommendation:** {{.Recommendation}}{{end}}

---
{{end}}

## Credential Inventory

| Username | Type | Host | Found |
|----------|------|------|-------|
{{range .Credentials}}| {{.Username}} | {{.SecretType}} | {{.Host}} | {{.FoundAt}} |
{{end}}

## Evidence Chain Integrity

Chain status: {{if .ChainIntact}}**INTACT**{{else}}**BROKEN**{{end}}
`

const findingCardTemplate = `# Finding: {{.Title}}

**Severity:** {{.Priority}}
**MITRE ATT&CK:** {{range .Mitre}}{{.}} {{end}}
**Status:** {{.Verified}}

## Description

{{.Description}}

## Evidence

{{range .EvidenceIDs}}
- Evidence #{{.}}
{{end}}

## Recommendation

{{.Recommendation}}
`
