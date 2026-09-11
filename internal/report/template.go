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
		"pct":       func(a, b int) int { if b == 0 { return 0 }; return a * 100 / b },
		"testedIcon": func(ok bool) string { if ok { return "✓" }; return "—" },
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
<title>{{.Data.Engagement.Name}} — Penetration Test Report</title>
<style>
:root {
  --accent: {{.Config.AccentColor}};
  --header: {{.Config.HeaderColor}};
  --bg: #fff;
  --fg: #1a1a2e;
  --muted: #6c757d;
  --border: #ddd;
  --surface: #f8f9fa;
}
*{box-sizing:border-box}
body { font-family: {{.Config.FontFamily}}; max-width: 1000px; margin: 0 auto; padding: 2rem; color: var(--fg); background: var(--bg); line-height: 1.7; font-size: 14px; }
h1 { color: var(--header); border-bottom: 3px solid var(--accent); padding-bottom: 0.5rem; font-size: 1.8rem; margin-top: 2.5rem; }
h2 { color: var(--header); margin-top: 2.5rem; border-bottom: 2px solid var(--border); padding-bottom: 0.4rem; font-size: 1.4rem; }
h3 { color: #0f3460; font-size: 1.15rem; margin-top: 1.5rem; }
h4 { color: var(--muted); font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; margin: 1rem 0 0.3rem; }

{{if .Config.CompanyName}}
.cover { text-align: center; padding: 3rem 2rem; margin-bottom: 2rem; border-bottom: 3px solid var(--accent); }
.cover h1 { border: none; font-size: 2.2rem; margin-bottom: 0.5rem; }
.cover .subtitle { font-size: 1.1rem; color: var(--muted); }
.cover .meta { margin-top: 1.5rem; font-size: 0.95rem; color: var(--fg); }
.cover .meta span { display: inline-block; margin: 0 1rem; }
{{end}}

.toc { background: var(--surface); border: 1px solid var(--border); border-radius: 6px; padding: 1.2rem 1.5rem; margin: 1.5rem 0; }
.toc h3 { margin: 0 0 0.8rem; color: var(--header); }
.toc ol { margin: 0; padding-left: 1.5rem; }
.toc li { margin: 0.3rem 0; }
.toc a { color: var(--header); text-decoration: none; }
.toc a:hover { text-decoration: underline; }

table { border-collapse: collapse; width: 100%; margin: 1rem 0; font-size: 0.9rem; }
th, td { border: 1px solid var(--border); padding: 8px 12px; text-align: left; }
th { background: var(--header); color: #fff; font-weight: 600; font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.03em; }
tr:nth-child(even) { background: var(--surface); }

.critical { color: #dc3545; font-weight: bold; }
.high { color: var(--accent); font-weight: bold; }
.medium { color: #f0a500; font-weight: bold; }
.low { color: #17a2b8; }
.info { color: var(--muted); }
.confirmed { color: #28a745; }
.false-positive { color: var(--muted); text-decoration: line-through; }

.stat-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.8rem; margin: 1.2rem 0; }
.stat-box { border: 2px solid var(--border); border-radius: 6px; padding: 0.8rem; text-align: center; }
.stat-box .number { font-size: 1.8rem; font-weight: bold; display: block; line-height: 1.2; }
.stat-box .label { font-size: 0.7rem; text-transform: uppercase; color: var(--muted); letter-spacing: 0.05em; }

.risk-bar { display: flex; height: 28px; border-radius: 4px; overflow: hidden; margin: 0.8rem 0; font-size: 0.75rem; font-weight: 600; }
.risk-bar span { display: flex; align-items: center; justify-content: center; color: #fff; min-width: 30px; }
.risk-bar .rb-crit { background: #dc3545; }
.risk-bar .rb-high { background: #e94560; }
.risk-bar .rb-med { background: #f0a500; }
.risk-bar .rb-low { background: #17a2b8; }
.risk-bar .rb-info { background: #6c757d; }

code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 0.85em; font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; }
pre { background: #1a1a2e; color: #e0e0e0; padding: 1rem; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; font-size: 0.85rem; line-height: 1.5; font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; }

.chain-ok { color: #28a745; font-weight: bold; }
.chain-fail { color: #dc3545; font-weight: bold; }

.finding { border: 1px solid var(--border); border-radius: 8px; padding: 1.2rem 1.5rem; margin: 1.2rem 0; border-left: 5px solid var(--border); page-break-inside: avoid; }
.finding.p-critical { border-left-color: #dc3545; }
.finding.p-high { border-left-color: var(--accent); }
.finding.p-medium { border-left-color: #f0a500; }
.finding.p-low { border-left-color: #17a2b8; }
.finding .f-header { display: flex; justify-content: space-between; align-items: baseline; flex-wrap: wrap; gap: 0.5rem; }
.finding .f-meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.3rem 1.5rem; margin: 0.8rem 0; padding: 0.6rem 0; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); font-size: 0.9rem; }
.finding .f-meta dt { font-weight: 600; color: var(--muted); font-size: 0.75rem; text-transform: uppercase; }
.finding .f-meta dd { margin: 0 0 0.3rem; }

.sev-badge { display: inline-block; padding: 2px 10px; border-radius: 3px; color: #fff; font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; }
.sev-critical { background: #dc3545; }
.sev-high { background: #e94560; }
.sev-medium { background: #f0a500; color: #1a1a2e; }
.sev-low { background: #17a2b8; }
.sev-info { background: #6c757d; }

.rec-box { margin: 0.8rem 0; padding: 10px 14px; background: #f0fff0; border-left: 3px solid #28a745; border-radius: 4px; }
.rec-box strong { color: #155724; }

.poc-section { margin: 0.8rem 0; }
.poc-section strong { display: block; margin-bottom: 0.3rem; }

.scope-bar { display: flex; align-items: center; gap: 0.8rem; margin: 0.5rem 0; }
.scope-bar .bar { flex: 1; height: 22px; background: var(--border); border-radius: 4px; overflow: hidden; }
.scope-bar .bar-fill { height: 100%; background: #28a745; border-radius: 4px 0 0 4px; transition: width 0.3s; }
.scope-bar .bar-label { font-size: 0.85rem; font-weight: 600; white-space: nowrap; }

.cred-table td:nth-child(2) { font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; }

.footer { margin-top: 3rem; padding-top: 1rem; border-top: 2px solid var(--border); color: var(--muted); font-size: 0.8rem; text-align: center; }

.disclaimer { background: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; padding: 0.8rem 1rem; margin: 1rem 0; font-size: 0.85rem; color: #856404; }

@media print {
  body { max-width: none; font-size: 12px; }
  .no-print { display: none; }
  .finding { page-break-inside: avoid; }
  .cover { page-break-after: always; }
}
</style>
</head>
<body>

{{if .Config.CompanyName}}
<div class="cover">
  {{if .Config.CompanyLogo}}<img src="{{.Config.CompanyLogo}}" alt="Logo" style="height:64px;margin-bottom:1rem">{{end}}
  <h1>Penetration Test Report</h1>
  <div class="subtitle">{{.Data.Engagement.Name}}</div>
  <div class="meta">
    {{if .Data.Engagement.Client}}<span><strong>Client:</strong> {{.Data.Engagement.Client}}</span>{{end}}
    <span><strong>Date:</strong> {{.Data.GeneratedAt}}</span>
    <span><strong>Prepared by:</strong> {{.Config.CompanyName}}</span>
  </div>
</div>
{{else}}
<h1 style="margin-top:0">{{.Data.Engagement.Name}} — Penetration Test Report</h1>
{{if .Data.Engagement.Client}}<p><strong>Client:</strong> {{.Data.Engagement.Client}} | <strong>Date:</strong> {{.Data.GeneratedAt}}</p>{{end}}
{{end}}

<div class="disclaimer">
  <strong>Confidential.</strong> This document contains sensitive security findings. Distribution is restricted to authorized personnel only. Unauthorized disclosure may compromise system security.
</div>

<div class="toc">
  <h3>Table of Contents</h3>
  <ol>
    {{if not .Options.TechOnly}}<li><a href="#executive-summary">Executive Summary</a></li>{{end}}
    <li><a href="#scope">Scope &amp; Coverage</a></li>
    {{if not .Options.ExecOnly}}
    <li><a href="#findings-detail">Findings — Technical Details</a></li>
    {{if .Data.Credentials}}<li><a href="#credentials">Compromised Credentials</a></li>{{end}}
    <li><a href="#timeline">Evidence Timeline</a></li>
    {{end}}
    <li><a href="#integrity">Evidence Integrity</a></li>
  </ol>
</div>

{{if not .Options.TechOnly}}
<h1 id="executive-summary">1. Executive Summary</h1>

<p>This report presents the findings of the penetration test conducted against <strong>{{.Data.Engagement.Name}}</strong>{{if .Data.Engagement.Client}} for <strong>{{.Data.Engagement.Client}}</strong>{{end}}. The assessment identified <strong>{{.Data.Stats.TotalFindings}} vulnerabilities</strong>, of which <strong class="critical">{{.Data.Stats.Critical}} are critical</strong> and <strong class="high">{{.Data.Stats.High}} are high severity</strong>.</p>

<div class="stat-grid">
  <div class="stat-box" style="border-color:#dc3545"><span class="number critical">{{.Data.Stats.Critical}}</span><span class="label">Critical</span></div>
  <div class="stat-box" style="border-color:#e94560"><span class="number high">{{.Data.Stats.High}}</span><span class="label">High</span></div>
  <div class="stat-box" style="border-color:#f0a500"><span class="number medium">{{.Data.Stats.Medium}}</span><span class="label">Medium</span></div>
  <div class="stat-box" style="border-color:#17a2b8"><span class="number low">{{.Data.Stats.Low}}</span><span class="label">Low</span></div>
</div>

{{if .Data.Stats.TotalFindings}}
<h4>Risk Distribution</h4>
<div class="risk-bar">
  {{if .Data.Stats.Critical}}<span class="rb-crit" style="flex:{{.Data.Stats.Critical}}">{{.Data.Stats.Critical}}</span>{{end}}
  {{if .Data.Stats.High}}<span class="rb-high" style="flex:{{.Data.Stats.High}}">{{.Data.Stats.High}}</span>{{end}}
  {{if .Data.Stats.Medium}}<span class="rb-med" style="flex:{{.Data.Stats.Medium}}">{{.Data.Stats.Medium}}</span>{{end}}
  {{if .Data.Stats.Low}}<span class="rb-low" style="flex:{{.Data.Stats.Low}}">{{.Data.Stats.Low}}</span>{{end}}
  {{if .Data.Stats.Info}}<span class="rb-info" style="flex:{{.Data.Stats.Info}}">{{.Data.Stats.Info}}</span>{{end}}
</div>
{{end}}

<h4>Key Metrics</h4>
<table>
<tr><th>Metric</th><th>Value</th></tr>
<tr><td>Total Findings</td><td><strong>{{.Data.Stats.TotalFindings}}</strong></td></tr>
<tr><td>Confirmed</td><td class="confirmed">{{.Data.Stats.Confirmed}}</td></tr>
<tr><td>False Positives</td><td>{{.Data.Stats.FalsePositive}}</td></tr>
<tr><td>Evidence Entries</td><td>{{.Data.Stats.TotalEvidence}}</td></tr>
<tr><td>Credentials Compromised</td><td>{{.Data.CredCount}}</td></tr>
<tr><td>Scope Coverage</td><td>{{.Data.ScopeTested}}/{{.Data.ScopeTotal}} hosts tested</td></tr>
<tr><td>Sessions</td><td>{{.Data.Stats.TotalSessions}}</td></tr>
</table>

<h2>Findings Summary</h2>
{{if .Data.Findings}}
<table>
<tr><th>#</th><th>Severity</th><th>Status</th><th>Finding</th><th>MITRE</th></tr>
{{range $i, $f := .Data.Findings}}
<tr>
  <td>{{add $i 1}}</td>
  <td><span class="sev-badge sev-{{$f.Priority}}">{{$f.Priority}}</span></td>
  <td class="{{$f.Verified}}">{{$f.Verified}}</td>
  <td><a href="#finding-{{add $i 1}}" style="color:var(--fg);text-decoration:none">{{$f.Title}}</a></td>
  <td>{{if $f.Mitre}}{{join $f.Mitre ", "}}{{end}}</td>
</tr>
{{end}}
</table>
{{else}}
<p>No findings to report.</p>
{{end}}
{{end}}

<h1 id="scope">{{if not .Options.TechOnly}}2. {{end}}Scope &amp; Coverage</h1>

{{if .Data.Scope}}
<div class="scope-bar">
  <div class="bar"><div class="bar-fill" style="width:{{if .Data.ScopeTotal}}{{pct .Data.ScopeTested .Data.ScopeTotal}}{{else}}0{{end}}%"></div></div>
  <div class="bar-label">{{.Data.ScopeTested}}/{{.Data.ScopeTotal}} ({{if .Data.ScopeTotal}}{{pct .Data.ScopeTested .Data.ScopeTotal}}{{else}}0{{end}}%)</div>
</div>
<table>
<tr><th>Host</th><th>Tested</th></tr>
{{range .Data.Scope}}
<tr><td><code>{{.Host}}</code></td><td>{{testedIcon .Tested}}</td></tr>
{{end}}
</table>
{{else}}
<p>No scope hosts defined for this engagement.</p>
{{end}}

{{if not .Options.ExecOnly}}
<h1 id="findings-detail">{{if not .Options.TechOnly}}3{{else}}2{{end}}. Findings — Technical Details</h1>

{{range $i, $f := .Data.Findings}}
<div class="finding p-{{$f.Priority}}" id="finding-{{add $i 1}}">
  <div class="f-header">
    <h3 style="margin:0">{{add $i 1}}. {{$f.Title}}</h3>
    <span class="sev-badge sev-{{$f.Priority}}">{{$f.Priority}}</span>
  </div>

  <dl class="f-meta">
    <div><dt>Severity</dt><dd class="{{$f.Priority}}">{{upper $f.Priority}}</dd></div>
    <div><dt>Status</dt><dd class="{{$f.Verified}}">{{$f.Verified}}</dd></div>
    {{if $f.Mitre}}<div><dt>MITRE ATT&amp;CK</dt><dd>{{join $f.Mitre ", "}}</dd></div>{{end}}
    {{if $f.VerifiedAt}}<div><dt>Verified</dt><dd>{{fmtTime $f.VerifiedAt}}</dd></div>{{end}}
  </dl>

  {{if $f.Description}}
  <h4>Description</h4>
  <p>{{$f.Description}}</p>
  {{end}}

  {{if $f.Notes}}
  <div class="poc-section">
    <h4>Proof of Concept</h4>
    <pre>{{$f.Notes}}</pre>
  </div>
  {{end}}

  {{if $f.Recommendation}}
  <div class="rec-box">
    <h4 style="margin-top:0;color:#155724">Recommendation</h4>
    <p style="margin:0">{{$f.Recommendation}}</p>
  </div>
  {{end}}
</div>
{{end}}

{{if .Data.Credentials}}
<h1 id="credentials">{{if not .Options.TechOnly}}4{{else}}3{{end}}. Compromised Credentials</h1>
<p>The following {{.Data.CredCount}} credentials were obtained during the assessment. Secrets are stored encrypted and not included in this report.</p>
<table class="cred-table">
<tr><th>#</th><th>Username / Key</th><th>Type</th><th>Host</th><th>Found</th></tr>
{{range $i, $c := .Data.Credentials}}
<tr>
  <td>{{add $i 1}}</td>
  <td><code>{{$c.Username}}</code></td>
  <td>{{$c.CredType}}</td>
  <td>{{if $c.Host}}<code>{{$c.Host}}</code>{{else}}—{{end}}</td>
  <td>{{fmtTime $c.FoundAt}}</td>
</tr>
{{end}}
</table>
{{end}}

<h1 id="timeline">{{if not .Options.TechOnly}}{{if .Data.Credentials}}5{{else}}4{{end}}{{else}}{{if .Data.Credentials}}4{{else}}3{{end}}{{end}}. Evidence Timeline</h1>
<p>{{.Data.Stats.TotalEvidence}} evidence entries captured with tamper-proof hash chain.</p>
<div style="overflow-x:auto">
<table>
<tr><th>#</th><th>Time</th><th>Type</th><th>Command / Action</th><th>Result (truncated)</th><th>Exit</th></tr>
{{range .Data.Evidence}}
<tr>
  <td>{{.ID}}</td>
  <td style="white-space:nowrap">{{fmtTime .Timestamp}}</td>
  <td>{{.Action}}</td>
  <td><code>{{truncate .Input 100}}</code></td>
  <td style="max-width:300px;overflow:hidden;text-overflow:ellipsis"><code>{{truncate .Output 150}}</code></td>
  <td>{{if eq .ExitCode 0}}{{.ExitCode}}{{else}}<span style="color:#dc3545">{{.ExitCode}}</span>{{end}}</td>
</tr>
{{end}}
</table>
</div>
{{end}}

<h1 id="integrity">Evidence Integrity</h1>
<p><strong>Hash Chain:</strong> <span class="{{chainClass .Data.ChainIntact}}">{{chainIcon .Data.ChainIntact}}</span></p>
<p style="font-size:0.85rem;color:var(--muted)">Each evidence entry is SHA-256 chained to its predecessor. An INTACT chain confirms no evidence has been tampered with, inserted, or deleted after capture.</p>

<div class="footer">
  <p>Generated by <strong>RT — Red Team Evidence Logger</strong></p>
  <p>{{.Data.GeneratedAt}} | Evidence chain: <span class="{{chainClass .Data.ChainIntact}}">{{chainIcon .Data.ChainIntact}}</span></p>
</div>
</body>
</html>`
