package evidence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/rt/internal/credentials"
	"github.com/user/rt/internal/rules"
)

// AutoFlagResult contains the results of auto-flagging an evidence entry.
type AutoFlagResult struct {
	Matches       []rules.Match
	CredsFound    []credentials.ParsedCred
	TagsAdded     []string
	PrioritySet   string
	MilestoneText string
}

// AutoFlag runs the rules engine and auto-cred parser on an evidence entry,
// then updates the evidence record with tags/priority/mitre.
func AutoFlag(db *sql.DB, ev *Evidence, engID, operator string) (*AutoFlagResult, error) {
	if ev.Output == "" {
		return nil, nil
	}

	matches := rules.MatchOutput(ev.Output)
	if len(matches) == 0 {
		return nil, nil
	}

	result := &AutoFlagResult{Matches: matches}

	// Collect new tags and priority
	newTags := rules.CollectTags(matches, ev.Tags)
	newPriority := rules.HighestPriority(matches, ev.Priority)

	// Collect MITRE IDs
	var mitreIDs []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if m.Mitre != "" && !seen[m.Mitre] {
			mitreIDs = append(mitreIDs, m.Mitre)
			seen[m.Mitre] = true
		}
	}

	// Update evidence record
	tagsJSON, _ := json.Marshal(newTags)
	mitreJSON, _ := json.Marshal(mitreIDs)

	_, err := db.Exec(
		`UPDATE evidence SET tags = ?, priority = ?, mitre = ? WHERE id = ?`,
		string(tagsJSON), newPriority, string(mitreJSON), ev.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update evidence flags: %w", err)
	}

	result.TagsAdded = newTags
	result.PrioritySet = newPriority

	// Auto-cred: extract and store credentials
	for _, m := range matches {
		if !m.AutoCred {
			continue
		}

		host := extractHost(ev.Input)
		parsedCreds := credentials.ParseCredentials(ev.Output, host)
		for _, pc := range parsedCreds {
			err := credentials.Store(db, engID, pc.Username, pc.Secret, pc.SecretType, pc.Host, operator, ev.ID)
			if err != nil {
				continue // best-effort
			}
			result.CredsFound = append(result.CredsFound, pc)
		}
		break // only parse once
	}

	// Auto-milestone
	for _, m := range matches {
		if m.AutoMilestone {
			result.MilestoneText = fmt.Sprintf("[auto] %s detected", m.RuleName)
			break
		}
	}

	return result, nil
}

// PrintAutoFlagResult prints auto-flag results to the console.
func PrintAutoFlagResult(r *AutoFlagResult) {
	if r == nil {
		return
	}

	for _, m := range r.Matches {
		icon := "+"
		color := "\033[33m" // yellow
		if m.Priority == "critical" {
			color = "\033[1;31m" // bold red
		}
		fmt.Printf("  %s[auto-flag] %s: %s [%s] %s\033[0m\n", color, icon, m.RuleName, m.Tag, m.Mitre)
	}

	if len(r.CredsFound) > 0 {
		fmt.Printf("  \033[1;33m[auto-cred] %d credential(s) extracted and stored\033[0m\n", len(r.CredsFound))
		for _, c := range r.CredsFound {
			masked := maskSecret(c.Secret)
			fmt.Printf("    %s : %s (%s)\n", c.Username, masked, c.SecretType)
		}
	}

	if r.MilestoneText != "" {
		fmt.Printf("  \033[1;32m[auto-milestone] %s\033[0m\n", r.MilestoneText)
	}
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func extractHost(input string) string {
	// Try to extract host/IP from command input
	parts := strings.Fields(input)
	for _, p := range parts {
		// Simple IP check
		if isIPLike(p) {
			return p
		}
	}
	// Check for -t or --target flags
	for i, p := range parts {
		if (p == "-t" || p == "--target" || p == "-target") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func isIPLike(s string) bool {
	dots := 0
	for _, c := range s {
		if c == '.' {
			dots++
		} else if c < '0' || c > '9' {
			if c == '/' { // CIDR
				continue
			}
			return false
		}
	}
	return dots == 3
}
