package report

import (
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/user/rt/internal/credentials"
	"github.com/user/rt/internal/engagement"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/findings"
	"github.com/user/rt/internal/session"
)

type ReportData struct {
	Engagement   *engagement.Engagement
	Sessions     []session.Session
	Findings     []findings.Finding
	Evidence     []evidence.Evidence
	CredCount    int
	GeneratedAt  string
	ChainIntact  bool
	Stats        Stats
}

type Stats struct {
	TotalEvidence   int
	TotalFindings   int
	TotalSessions   int
	Confirmed       int
	FalsePositive   int
	Unverified      int
	Critical        int
	High            int
	Medium          int
	Low             int
	Info            int
}

type Options struct {
	ExecOnly     bool
	TechOnly     bool
	CriticalOnly bool
	VerifiedOnly bool
}

func Gather(db *sql.DB, engID string, opts Options) (*ReportData, error) {
	eng, err := engagement.Get(db, engID)
	if err != nil {
		return nil, fmt.Errorf("engagement not found: %w", err)
	}

	sessions, err := session.List(db, engID)
	if err != nil {
		return nil, err
	}

	allFindings, err := findings.List(db, engID)
	if err != nil {
		return nil, err
	}

	var filtered []findings.Finding
	for _, f := range allFindings {
		if opts.VerifiedOnly && f.Verified != "confirmed" {
			continue
		}
		if opts.CriticalOnly && f.Priority != "critical" && f.Priority != "high" {
			continue
		}
		filtered = append(filtered, f)
	}

	ev, err := evidence.Timeline(db, engID, 10000)
	if err != nil {
		return nil, err
	}

	credCount := 0
	creds, _ := credentials.List(db, engID)
	credCount = len(creds)

	chainResults, _ := evidence.VerifyChain(db, engID)
	chainOK := true
	for _, r := range chainResults {
		if !r.Intact {
			chainOK = false
			break
		}
	}

	uv, cf, fp := findings.CountByStatus(db, engID)

	stats := Stats{
		TotalEvidence: len(ev),
		TotalFindings: len(allFindings),
		TotalSessions: len(sessions),
		Confirmed:     cf,
		FalsePositive: fp,
		Unverified:    uv,
	}
	for _, f := range allFindings {
		switch f.Priority {
		case "critical":
			stats.Critical++
		case "high":
			stats.High++
		case "medium":
			stats.Medium++
		case "low":
			stats.Low++
		default:
			stats.Info++
		}
	}

	return &ReportData{
		Engagement:  eng,
		Sessions:    sessions,
		Findings:    filtered,
		Evidence:    ev,
		CredCount:   credCount,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ChainIntact: chainOK,
		Stats:       stats,
	}, nil
}

func RenderMarkdown(data *ReportData, opts Options) string {
	var b strings.Builder

	if opts.ExecOnly {
		renderExecMD(&b, data)
	} else if opts.TechOnly {
		renderTechMD(&b, data)
	} else {
		renderExecMD(&b, data)
		b.WriteString("\n---\n\n")
		renderTechMD(&b, data)
	}

	return b.String()
}

func RenderHTML(data *ReportData, opts Options) string {
	result, err := RenderTemplateHTML(data, opts, nil)
	if err != nil {
		return fmt.Sprintf("<html><body><h1>Template error</h1><pre>%s</pre></body></html>", h(err.Error()))
	}
	return result
}

func renderExecMD(b *strings.Builder, data *ReportData) {
	b.WriteString(fmt.Sprintf("# %s — Executive Summary\n\n", data.Engagement.Name))
	if data.Engagement.Client != "" {
		b.WriteString(fmt.Sprintf("**Client:** %s\n\n", data.Engagement.Client))
	}
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", data.GeneratedAt))

	b.WriteString("## Overview\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Count |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| Total Findings | %d |\n", data.Stats.TotalFindings))
	b.WriteString(fmt.Sprintf("| Critical | %d |\n", data.Stats.Critical))
	b.WriteString(fmt.Sprintf("| High | %d |\n", data.Stats.High))
	b.WriteString(fmt.Sprintf("| Medium | %d |\n", data.Stats.Medium))
	b.WriteString(fmt.Sprintf("| Low | %d |\n", data.Stats.Low))
	b.WriteString(fmt.Sprintf("| Confirmed | %d |\n", data.Stats.Confirmed))
	b.WriteString(fmt.Sprintf("| False Positive | %d |\n", data.Stats.FalsePositive))
	b.WriteString(fmt.Sprintf("| Evidence Entries | %d |\n", data.Stats.TotalEvidence))
	b.WriteString(fmt.Sprintf("| Sessions | %d |\n", data.Stats.TotalSessions))
	b.WriteString(fmt.Sprintf("| Credentials Found | %d |\n\n", data.CredCount))

	b.WriteString("## Findings Summary\n\n")
	if len(data.Findings) == 0 {
		b.WriteString("No findings to report.\n\n")
	} else {
		b.WriteString("| # | Priority | Status | Title |\n|---|---|---|---|\n")
		for i, f := range data.Findings {
			b.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", i+1, f.Priority, f.Verified, f.Title))
		}
		b.WriteString("\n")
	}

	chain := "INTACT"
	if !data.ChainIntact {
		chain = "BROKEN"
	}
	b.WriteString(fmt.Sprintf("**Evidence Chain Integrity:** %s\n\n", chain))
}

func renderTechMD(b *strings.Builder, data *ReportData) {
	b.WriteString("# Technical Details\n\n")

	for i, f := range data.Findings {
		b.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, f.Title))
		b.WriteString(fmt.Sprintf("**Priority:** %s | **Status:** %s\n\n", f.Priority, f.Verified))
		if f.Description != "" {
			b.WriteString(fmt.Sprintf("**Description:** %s\n\n", f.Description))
		}
		if len(f.Mitre) > 0 {
			b.WriteString(fmt.Sprintf("**MITRE ATT&CK:** %s\n\n", strings.Join(f.Mitre, ", ")))
		}
		if f.Recommendation != "" {
			b.WriteString(fmt.Sprintf("**Recommendation:** %s\n\n", f.Recommendation))
		}
		if len(f.EvidenceIDs) > 0 {
			b.WriteString(fmt.Sprintf("**Evidence IDs:** %v\n\n", f.EvidenceIDs))
		}
		if f.Notes != "" {
			b.WriteString(fmt.Sprintf("**Notes:** %s\n\n", f.Notes))
		}
	}

	b.WriteString("## Timeline\n\n")
	b.WriteString("| Time | Action | Input | Exit |\n|---|---|---|---|\n")
	for _, e := range data.Evidence {
		ts := fmtTime(e.Timestamp)
		input := e.Input
		if len(input) > 60 {
			input = input[:57] + "..."
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n", ts, e.Action, input, e.ExitCode))
	}
	b.WriteString("\n")
}


func h(s string) string {
	return html.EscapeString(s)
}

