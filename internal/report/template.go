package report

import (
	"bytes"
	"fmt"
	"text/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/rt/internal/config"
)

type TemplateConfig struct {
	Name        string `yaml:"name"`
	CompanyName string `yaml:"company_name"`
	CompanyLogo string `yaml:"company_logo"`
	AccentColor string `yaml:"accent_color"`
	HeaderColor string `yaml:"header_color"`
	FontFamily  string `yaml:"font_family"`
}

var defaultTemplate = TemplateConfig{
	Name:        "default",
	AccentColor: "#e94560",
	HeaderColor: "#16213e",
	FontFamily:  "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
}

func LoadTemplate(name string) (*TemplateConfig, error) {
	if name == "" || name == "default" {
		return &defaultTemplate, nil
	}

	path := filepath.Join(config.Home(), "templates", name+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", name, err)
	}

	cfg := defaultTemplate
	cfg.Name = name
	if err := parseYAML(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func parseYAML(data []byte, cfg *TemplateConfig) error {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")
		switch key {
		case "company_name":
			cfg.CompanyName = val
		case "company_logo":
			cfg.CompanyLogo = val
		case "accent_color":
			cfg.AccentColor = val
		case "header_color":
			cfg.HeaderColor = val
		case "font_family":
			cfg.FontFamily = val
		}
	}
	return nil
}

func RenderTemplateHTML(data *ReportData, opts Options, tmplCfg *TemplateConfig) (string, error) {
	if tmplCfg == nil {
		tmplCfg = &defaultTemplate
	}

	// Check for custom HTML template
	customPath := filepath.Join(config.Home(), "templates", tmplCfg.Name+".html")
	if tmplContent, err := os.ReadFile(customPath); err == nil {
		return renderCustomTemplate(string(tmplContent), data, opts, tmplCfg)
	}

	return renderBuiltinTemplate(data, opts, tmplCfg)
}

func renderCustomTemplate(tmplContent string, data *ReportData, opts Options, cfg *TemplateConfig) (string, error) {
	funcMap := template.FuncMap{
		"fmtTime":  fmtTime,
		"truncate": truncateStr,
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"join":     strings.Join,
		"add":      func(a, b int) int { return a + b },
		"pct":      func(a, b int) int { if b == 0 { return 0 }; return a * 100 / b },
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	tplData := map[string]interface{}{
		"Data":    data,
		"Options": opts,
		"Config":  cfg,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, tplData); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return buf.String(), nil
}

func renderBuiltinTemplate(data *ReportData, opts Options, cfg *TemplateConfig) (string, error) {
	funcMap := template.FuncMap{
		"fmtTime":   fmtTime,
		"truncate":  truncateStr,
		"upper":     strings.ToUpper,
		"chainIcon": func(ok bool) string { if ok { return "INTACT" }; return "BROKEN" },
		"chainClass": func(ok bool) string { if ok { return "chain-ok" }; return "chain-fail" },
		"join":      strings.Join,
		"add":       func(a, b int) int { return a + b },
	}

	t, err := template.New("report").Funcs(funcMap).Parse(builtinHTMLTemplate)
	if err != nil {
		return "", err
	}

	tplData := struct {
		Data    *ReportData
		Options Options
		Config  *TemplateConfig
	}{data, opts, cfg}

	var buf bytes.Buffer
	if err := t.Execute(&buf, tplData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fmtTime(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ts
		}
	}
	return t.Format("2006-01-02 15:04:05")
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

const builtinHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Data.Engagement.Name}} — Red Team Report</title>
<style>
:root {
  --accent: {{.Config.AccentColor}};
  --header: {{.Config.HeaderColor}};
}
body { font-family: {{.Config.FontFamily}}; max-width: 960px; margin: 0 auto; padding: 2rem; color: #1a1a2e; background: #fff; line-height: 1.6; }
h1 { color: var(--header); border-bottom: 3px solid var(--accent); padding-bottom: 0.5rem; }
h2 { color: var(--header); margin-top: 2rem; border-bottom: 1px solid #ddd; padding-bottom: 0.3rem; }
h3 { color: #0f3460; }
.branding { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.branding img { height: 48px; }
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
th { background: var(--header); color: #fff; }
tr:nth-child(even) { background: #f8f8f8; }
.critical { color: #dc3545; font-weight: bold; }
.high { color: var(--accent); font-weight: bold; }
.medium { color: #f0a500; }
.low { color: #17a2b8; }
.info { color: #6c757d; }
.confirmed { color: #28a745; }
.false-positive { color: #6c757d; text-decoration: line-through; }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 1rem; margin: 1rem 0; }
.stat-box { border: 2px solid #ddd; border-radius: 8px; padding: 1rem; text-align: center; }
.stat-box .number { font-size: 2rem; font-weight: bold; display: block; }
.stat-box .label { font-size: 0.8rem; text-transform: uppercase; color: #666; }
code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
pre { background: #1a1a2e; color: #e0e0e0; padding: 1rem; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
.chain-ok { color: #28a745; font-weight: bold; }
.chain-fail { color: #dc3545; font-weight: bold; }
.footer { margin-top: 3rem; padding-top: 1rem; border-top: 1px solid #ddd; color: #999; font-size: 0.8rem; }
.finding { border: 1px solid #ddd; border-radius: 8px; padding: 1rem 1.5rem; margin: 1rem 0; border-left: 4px solid #ddd; }
.finding.p-critical { border-left-color: #dc3545; }
.finding.p-high { border-left-color: var(--accent); }
.finding.p-medium { border-left-color: #f0a500; }
.finding.p-low { border-left-color: #17a2b8; }
@media print { body { max-width: none; } .no-print { display: none; } }
</style>
</head>
<body>

{{if .Config.CompanyName}}
<div class="branding">
  {{if .Config.CompanyLogo}}<img src="{{.Config.CompanyLogo}}" alt="Logo">{{end}}
  <span style="font-size:1.1rem;color:#666">{{.Config.CompanyName}}</span>
</div>
{{end}}

{{if not .Options.TechOnly}}
<h1>{{.Data.Engagement.Name}} — Executive Summary</h1>
{{if .Data.Engagement.Client}}<p><strong>Client:</strong> {{.Data.Engagement.Client}}</p>{{end}}
<p><strong>Generated:</strong> {{.Data.GeneratedAt}}</p>

<div class="stat-grid">
  <div class="stat-box"><span class="number critical">{{.Data.Stats.Critical}}</span><span class="label">Critical</span></div>
  <div class="stat-box"><span class="number high">{{.Data.Stats.High}}</span><span class="label">High</span></div>
  <div class="stat-box"><span class="number medium">{{.Data.Stats.Medium}}</span><span class="label">Medium</span></div>
  <div class="stat-box"><span class="number low">{{.Data.Stats.Low}}</span><span class="label">Low</span></div>
  <div class="stat-box"><span class="number">{{.Data.Stats.TotalFindings}}</span><span class="label">Findings</span></div>
  <div class="stat-box"><span class="number confirmed">{{.Data.Stats.Confirmed}}</span><span class="label">Confirmed</span></div>
  <div class="stat-box"><span class="number">{{.Data.Stats.TotalEvidence}}</span><span class="label">Evidence</span></div>
  <div class="stat-box"><span class="number">{{.Data.CredCount}}</span><span class="label">Credentials</span></div>
</div>

<h2>Findings Summary</h2>
{{if .Data.Findings}}
<table>
<tr><th>#</th><th>Priority</th><th>Status</th><th>Title</th></tr>
{{range $i, $f := .Data.Findings}}
<tr><td>{{add $i 1}}</td><td class="{{$f.Priority}}">{{$f.Priority}}</td><td class="{{$f.Verified}}">{{$f.Verified}}</td><td>{{$f.Title}}</td></tr>
{{end}}
</table>
{{else}}
<p>No findings to report.</p>
{{end}}

<p><strong>Evidence Chain Integrity:</strong> <span class="{{chainClass .Data.ChainIntact}}">{{chainIcon .Data.ChainIntact}}</span></p>
{{end}}

{{if not .Options.ExecOnly}}
<h2>Technical Details</h2>
{{range $i, $f := .Data.Findings}}
<div class="finding p-{{$f.Priority}}">
  <h3>{{add $i 1}}. {{$f.Title}}</h3>
  <p><strong>Priority:</strong> <span class="{{$f.Priority}}">{{$f.Priority}}</span> | <strong>Status:</strong> <span class="{{$f.Verified}}">{{$f.Verified}}</span></p>
  {{if $f.Description}}<p><strong>Description:</strong> {{$f.Description}}</p>{{end}}
  {{if $f.Mitre}}<p><strong>MITRE ATT&CK:</strong> {{join $f.Mitre ", "}}</p>{{end}}
  {{if $f.Recommendation}}<p><strong>Recommendation:</strong> {{$f.Recommendation}}</p>{{end}}
  {{if $f.Notes}}<p><strong>Notes:</strong> {{$f.Notes}}</p>{{end}}
</div>
{{end}}

<h2>Timeline</h2>
<table>
<tr><th>Time</th><th>Action</th><th>Input</th><th>Output</th><th>Exit</th></tr>
{{range .Data.Evidence}}
<tr><td>{{fmtTime .Timestamp}}</td><td>{{.Action}}</td><td><code>{{truncate .Input 80}}</code></td><td><code>{{truncate .Output 120}}</code></td><td>{{.ExitCode}}</td></tr>
{{end}}
</table>
{{end}}

<div class="footer">
  Generated by RT — Red Team Evidence Logger | {{.Data.GeneratedAt}} | Evidence chain: <span class="{{chainClass .Data.ChainIntact}}">{{chainIcon .Data.ChainIntact}}</span>
</div>
</body>
</html>`
