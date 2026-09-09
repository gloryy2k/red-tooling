package export

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/findings"
)

// GhostwriterFinding matches Ghostwriter's expected import format.
type GhostwriterFinding struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Impact         string   `json:"impact"`
	Recommendation string   `json:"recommendation"`
	Severity       string   `json:"severity"`
	MitreTechnique []string `json:"mitre_technique,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
}

func ExportGhostwriter(db *sql.DB, engID, operator string) (string, error) {
	list, err := findings.List(db, engID)
	if err != nil {
		return "", err
	}

	var gFindings []GhostwriterFinding
	for _, f := range list {
		if f.Verified == "false-positive" {
			continue
		}

		var evStrings []string
		for _, eid := range f.EvidenceIDs {
			ev, err := evidence.Get(db, eid)
			if err == nil {
				evStrings = append(evStrings, fmt.Sprintf("[%s] %s → %s", ev.Timestamp, ev.Input, truncOutput(ev.Output, 200)))
			}
		}

		gFindings = append(gFindings, GhostwriterFinding{
			Title:          f.Title,
			Description:    f.Description,
			Impact:         mapSeverity(f.Priority),
			Recommendation: f.Recommendation,
			Severity:       mapSeverity(f.Priority),
			MitreTechnique: f.Mitre,
			Evidence:       evStrings,
		})
	}

	data, err := json.MarshalIndent(gFindings, "", "  ")
	if err != nil {
		return "", err
	}

	audit.Log(db, operator, "report.export", "export", "ghostwriter",
		map[string]int{"findings": len(gFindings)})

	return string(data), nil
}

func ExportJSON(db *sql.DB, engID, operator string) (string, error) {
	list, err := findings.List(db, engID)
	if err != nil {
		return "", err
	}

	ev, err := evidence.Timeline(db, engID, 10000)
	if err != nil {
		return "", err
	}

	out := map[string]interface{}{
		"engagement_id": engID,
		"findings":      list,
		"evidence":      ev,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}

	audit.Log(db, operator, "report.export", "export", "json",
		map[string]int{"findings": len(list), "evidence": len(ev)})

	return string(data), nil
}

func ExportCSV(db *sql.DB, engID, operator string) (string, error) {
	list, err := findings.List(db, engID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	w := csv.NewWriter(&b)
	w.Write([]string{"ID", "Title", "Priority", "Status", "Description", "Recommendation", "MITRE", "Evidence_IDs"})

	for _, f := range list {
		var evIDs []string
		for _, eid := range f.EvidenceIDs {
			evIDs = append(evIDs, fmt.Sprintf("%d", eid))
		}
		w.Write([]string{
			fmt.Sprintf("%d", f.ID),
			f.Title,
			f.Priority,
			f.Verified,
			f.Description,
			f.Recommendation,
			strings.Join(f.Mitre, ";"),
			strings.Join(evIDs, ";"),
		})
	}
	w.Flush()

	audit.Log(db, operator, "report.export", "export", "csv",
		map[string]int{"findings": len(list)})

	return b.String(), nil
}

// MITRELayer generates a MITRE ATT&CK Navigator layer JSON.
type MITRELayer struct {
	Name        string           `json:"name"`
	Version     string           `json:"versions"`
	Domain      string           `json:"domain"`
	Description string           `json:"description"`
	Techniques  []MITRETechnique `json:"techniques"`
}

type MITRETechnique struct {
	TechniqueID string `json:"techniqueID"`
	Score       int    `json:"score"`
	Color       string `json:"color"`
	Comment     string `json:"comment"`
}

func ExportMITRE(db *sql.DB, engID, operator string) (string, error) {
	list, err := findings.List(db, engID)
	if err != nil {
		return "", err
	}

	techMap := make(map[string][]string)
	techPri := make(map[string]string)

	for _, f := range list {
		if f.Verified == "false-positive" {
			continue
		}
		for _, m := range f.Mitre {
			techMap[m] = append(techMap[m], f.Title)
			if priorityRank(f.Priority) > priorityRank(techPri[m]) {
				techPri[m] = f.Priority
			}
		}
	}

	var techniques []MITRETechnique
	for tid, titles := range techMap {
		techniques = append(techniques, MITRETechnique{
			TechniqueID: tid,
			Score:       priorityScore(techPri[tid]),
			Color:       priorityHex(techPri[tid]),
			Comment:     strings.Join(titles, "; "),
		})
	}

	layer := MITRELayer{
		Name:        fmt.Sprintf("RT - %s", engID),
		Version:     "4.5",
		Domain:      "enterprise-attack",
		Description: fmt.Sprintf("Auto-generated from RT engagement %s", engID),
		Techniques:  techniques,
	}

	data, err := json.MarshalIndent(layer, "", "  ")
	if err != nil {
		return "", err
	}

	audit.Log(db, operator, "report.export", "export", "mitre",
		map[string]int{"techniques": len(techniques)})

	return string(data), nil
}

func mapSeverity(priority string) string {
	switch priority {
	case "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return "Informational"
	}
}

func priorityRank(p string) int {
	switch p {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func priorityScore(p string) int {
	switch p {
	case "critical":
		return 100
	case "high":
		return 75
	case "medium":
		return 50
	case "low":
		return 25
	default:
		return 10
	}
}

func priorityHex(p string) string {
	switch p {
	case "critical":
		return "#dc3545"
	case "high":
		return "#e94560"
	case "medium":
		return "#f0a500"
	case "low":
		return "#17a2b8"
	default:
		return "#6c757d"
	}
}

func truncOutput(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
