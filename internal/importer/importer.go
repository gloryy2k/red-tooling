package importer

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

type ImportResult struct {
	Source  string
	Format  string
	Entries int
}

// Import detects format and imports accordingly.
func Import(db *sql.DB, engID, filePath, operator string) (*ImportResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	sessID, _ := crypto.RandomHex(8)
	sessName := fmt.Sprintf("import-%s-%s", filepath.Base(filePath), time.Now().Format("150405"))
	if err := session.Create(db, sessID, engID, sessName, "import:"+ext[1:], ""); err != nil {
		return nil, fmt.Errorf("create import session: %w", err)
	}

	var count int

	switch ext {
	case ".xml":
		count, err = importNmapXML(db, sessID, engID, operator, data)
	case ".json":
		count, err = importNucleiJSON(db, sessID, engID, operator, data)
	case ".csv":
		count, err = importCSV(db, sessID, engID, operator, data)
	default:
		count, err = importRawText(db, sessID, engID, operator, filePath, data)
	}

	if err != nil {
		return nil, err
	}

	session.Stop(db, sessID, operator)

	audit.Log(db, operator, "import.complete", "session", sessID,
		map[string]interface{}{"file": filepath.Base(filePath), "format": ext, "entries": count})

	return &ImportResult{
		Source:  filepath.Base(filePath),
		Format:  ext,
		Entries: count,
	}, nil
}

// Nmap XML structures
type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}
type nmapHost struct {
	Address  nmapAddr    `xml:"address"`
	Ports    nmapPorts   `xml:"ports"`
	Hostnames nmapNames  `xml:"hostnames"`
}
type nmapAddr struct {
	Addr string `xml:"addr,attr"`
}
type nmapPorts struct {
	Ports []nmapPort `xml:"port"`
}
type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   string      `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}
type nmapState struct {
	State string `xml:"state,attr"`
}
type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}
type nmapNames struct {
	Names []nmapHostname `xml:"hostname"`
}
type nmapHostname struct {
	Name string `xml:"name,attr"`
}

func importNmapXML(db *sql.DB, sessID, engID, operator string, data []byte) (int, error) {
	var run nmapRun
	if err := xml.Unmarshal(data, &run); err != nil {
		return 0, fmt.Errorf("parse nmap XML: %w", err)
	}

	count := 0
	for _, host := range run.Hosts {
		var lines []string
		addr := host.Address.Addr
		lines = append(lines, fmt.Sprintf("Host: %s", addr))

		for _, p := range host.Ports.Ports {
			if p.State.State != "open" {
				continue
			}
			svc := p.Service.Name
			if p.Service.Product != "" {
				svc += " " + p.Service.Product
			}
			if p.Service.Version != "" {
				svc += " " + p.Service.Version
			}
			lines = append(lines, fmt.Sprintf("  %s/%s open %s", p.PortID, p.Protocol, svc))
		}

		if len(lines) > 1 {
			output := strings.Join(lines, "\n")
			_, err := evidence.Insert(db, sessID, "import:nmap", "nmap scan "+addr, output, 0, 0, "", []string{"recon", "nmap"}, "", operator)
			if err == nil {
				count++
			}
		}
	}
	return count, nil
}

// Nuclei JSON line format
type nucleiResult struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name        string   `json:"name"`
		Severity    string   `json:"severity"`
		Tags        []string `json:"tags"`
		Description string   `json:"description"`
	} `json:"info"`
	MatcherName string `json:"matcher-name"`
	Host        string `json:"host"`
	Matched     string `json:"matched-at"`
}

func importNucleiJSON(db *sql.DB, sessID, engID, operator string, data []byte) (int, error) {
	count := 0
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var r nucleiResult
		if err := dec.Decode(&r); err != nil {
			if err == io.EOF {
				break
			}
			// Try as array
			var arr []nucleiResult
			if err2 := json.Unmarshal(data, &arr); err2 == nil {
				for _, r := range arr {
					insertNucleiResult(db, sessID, operator, r)
					count++
				}
				return count, nil
			}
			break
		}
		insertNucleiResult(db, sessID, operator, r)
		count++
	}
	return count, nil
}

func insertNucleiResult(db *sql.DB, sessID, operator string, r nucleiResult) {
	output := fmt.Sprintf("[%s] %s\nHost: %s\nMatched: %s",
		r.Info.Severity, r.Info.Name, r.Host, r.Matched)
	if r.Info.Description != "" {
		output += "\n" + r.Info.Description
	}

	tags := append([]string{"vuln-scan", "nuclei"}, r.Info.Tags...)
	priority := mapNucleiSeverity(r.Info.Severity)

	evidence.Insert(db, sessID, "import:nuclei", r.TemplateID+" "+r.Host, output, 0, 0, "", tags, priority, operator)
}

func mapNucleiSeverity(sev string) string {
	switch strings.ToLower(sev) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "info"
	}
}

func importCSV(db *sql.DB, sessID, engID, operator string, data []byte) (int, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) < 2 {
		return 0, fmt.Errorf("CSV has no data rows")
	}

	headers := records[0]
	count := 0
	for _, row := range records[1:] {
		var parts []string
		for i, h := range headers {
			if i < len(row) {
				parts = append(parts, fmt.Sprintf("%s: %s", h, row[i]))
			}
		}
		output := strings.Join(parts, "\n")
		_, err := evidence.Insert(db, sessID, "import:csv", "csv row", output, 0, 0, "", []string{"import"}, "", operator)
		if err == nil {
			count++
		}
	}
	return count, nil
}

func importRawText(db *sql.DB, sessID, engID, operator, filePath string, data []byte) (int, error) {
	output := string(data)
	if len(output) > 1024*1024 {
		output = output[:1024*1024] + "\n[truncated at 1MB]"
	}
	_, err := evidence.Insert(db, sessID, "import:text", filepath.Base(filePath), output, 0, 0, "", []string{"import"}, "", operator)
	if err != nil {
		return 0, err
	}
	return 1, nil
}
