package agentctx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/rt/internal/config"
)

func EngagementDir(engName string) string {
	return filepath.Join(config.Home(), "engagements", engName)
}

func GenerateContext(engName, client, scope string) error {
	dir := EngagementDir(engName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := writeCLAUDEMD(dir, engName, client, scope); err != nil {
		return err
	}

	if err := writeSkills(dir, engName); err != nil {
		return err
	}

	return nil
}

func writeCLAUDEMD(dir, engName, client, scope string) error {
	engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
	date := time.Now().Format("2006-01-02")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s — Red Team Engagement\n\n", engName))

	if client != "" {
		b.WriteString(fmt.Sprintf("**Client:** %s\n", client))
	}
	b.WriteString(fmt.Sprintf("**Engagement ID:** %s\n", engID))
	b.WriteString(fmt.Sprintf("**Start Date:** %s\n\n", date))

	if scope != "" {
		b.WriteString("## Scope\n\n")
		b.WriteString("```\n")
		for _, s := range strings.Split(scope, ",") {
			b.WriteString(strings.TrimSpace(s) + "\n")
		}
		b.WriteString("```\n\n")
	}

	b.WriteString(`## RT Tool — Quick Reference

All commands use the ` + "`rt`" + ` binary. The database must be unlocked first.

### Setup
` + "```bash" + `
rt unlock                              # decrypt database (enter passphrase)
` + "```" + `

### Evidence Capture
` + "```bash" + `
rt exec <command>                      # run command + auto-capture evidence
rt start                               # start interactive capture session
rt stop                                # stop session + lock db
rt tag "description"                   # tag current evidence
rt note "free text"                    # add a note
rt milestone "achievement"             # record milestone
rt bookmark "come back later"          # bookmark for follow-up
` + "```" + `

### Findings & Credentials
` + "```bash" + `
rt finding "Title" --priority critical --mitre T1190
rt verify-finding <id> confirmed
rt recommend <id> "Fix description"
rt cred <user> <secret> --host <ip>    # store credential
rt findings                            # list findings
rt creds                               # list credentials
` + "```" + `

### Scope & Progress
` + "```bash" + `
rt scope 10.0.0.1,10.0.0.2            # add hosts to scope
rt scope-tested 10.0.0.1              # mark host as tested
rt scope-untested                      # list untested hosts
rt checklist-load ptes                 # load PTES checklist
rt check <id>                          # check off item
rt standup                             # daily progress summary
` + "```" + `

### Import & Search
` + "```bash" + `
rt import scan.xml                     # import nmap/nuclei/CSV
rt search "keyword" --tag recon        # search evidence
rt query "SELECT * FROM evidence LIMIT 10"
` + "```" + `

### Reports & Export
` + "```bash" + `
rt report --html -o report.html --template default
rt export ghostwriter -o findings.json
rt export csv -o findings.csv
rt export mitre -o navigator.json
` + "```" + `

### Playbooks (Agent Mode)
` + "```bash" + `
rt agent-run recon --target 10.10.10.0/24
rt agent-run web-recon --target https://target.com
rt agent-run ad-recon --target 10.10.10.1 --domain acme.local
rt agent-run kerberos-attacks --target <dc> --domain <dom> --var username=<u> --var password=<p>
rt agent-run post-exploit --target <ip> --var username=<u> --var password=<p>
rt agent-run lateral-movement --target <subnet> --var username=<u> --var password=<p>
rt agent-run internal-enum --target <subnet>
rt agent-run bloodhound --target <dc> --var domain=<d> --var username=<u> --var password=<p>
rt agent-run cleanup --target <ip>
rt agent-exec "nmap -sV 10.10.10.1"   # single command as agent
rt agent-list                          # list available playbooks
` + "```" + `

### Dashboard
` + "```bash" + `
rt serve --listen localhost:8080       # start web dashboard
` + "```" + `

## Workflow for Agents

1. **Always capture evidence** — use ` + "`rt exec <cmd>`" + ` instead of raw commands
2. **Auto-flag** handles tagging — RT detects credentials, admin access, vulns, etc.
3. **Create findings** when you discover vulnerabilities — include priority + MITRE
4. **Mark scope** as tested after scanning each host
5. **Use playbooks** for structured phases (recon, enum, exploit, post-exploit)
6. **Check progress** with ` + "`rt standup`" + ` and ` + "`rt scope-untested`" + `
7. **Generate report** when phase is complete

## Auto-Flag Engine

RT automatically detects and tags based on command output:
- **Credentials** — password hashes, NTLM, TGT, valid creds
- **Admin access** — Domain Admin, SYSTEM, root
- **ADCS vulns** — ESC1-9, vulnerable certificates
- **Kerberoast/AS-REP** — Kerberos ticket hashes
- **SQLi/RCE** — injection findings
- **Lateral movement** — psexec, wmiexec, pivot indicators
- **Persistence** — scheduled tasks, services, registry keys
- **Named exploits** — ZeroLogon, PrintNightmare, etc.

Credentials from secretsdump/mimikatz/kerberoast output are auto-extracted and stored.
`)

	return os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(b.String()), 0600)
}

func writeSkills(dir, engName string) error {
	skillsDir := filepath.Join(dir, ".claude", "commands")
	if err := os.MkdirAll(skillsDir, 0700); err != nil {
		return err
	}

	skills := map[string]string{
		"recon.md": `# Run Reconnaissance

Execute network recon against the target scope.

1. Check scope: ` + "`rt scope-list`" + `
2. Pick untested hosts: ` + "`rt scope-untested`" + `
3. Run playbook: ` + "`rt agent-run recon --target <ip/cidr>`" + `
4. Or manual: ` + "`rt exec nmap -sV -sC <target>`" + `
5. Import results: ` + "`rt import {work_dir}/nmap.xml`" + `
6. Mark hosts tested: ` + "`rt scope-tested <ip>`" + `
7. Check checklist: ` + "`rt checklist`" + `
`,
		"exploit.md": `# Exploit Target

Run exploitation against identified vulnerabilities.

1. Review findings: ` + "`rt findings`" + `
2. For each vuln, use rt exec to capture the exploit:
   ` + "`rt exec <exploit-command>`" + `
3. On success, create finding:
   ` + "`rt finding \"Title\" --priority critical --mitre T1190`" + `
4. Store any creds found:
   ` + "`rt cred <user> <pass> --host <ip>`" + `
5. Verify findings:
   ` + "`rt verify-finding <id> confirmed`" + `
6. Add recommendations:
   ` + "`rt recommend <id> \"Remediation steps\"`" + `
`,
		"post-exploit.md": `# Post-Exploitation

Run post-exploitation on compromised hosts.

1. Use playbook: ` + "`rt agent-run post-exploit --target <ip> --var username=<u> --var password=<p>`" + `
2. Check for persistence: ` + "`rt agent-run persistence-check --target <ip> --var username=<u> --var password=<p>`" + `
3. Lateral movement: ` + "`rt agent-run lateral-movement --target <subnet> --var username=<u> --var password=<p>`" + `
4. AD attacks: ` + "`rt agent-run kerberos-attacks --target <dc> --var domain=<d>`" + `
5. Create findings for each escalation path found
6. Update scope: ` + "`rt scope-tested <new-ip>`" + `
`,
		"report.md": `# Generate Report

Generate the engagement report.

1. Review all findings: ` + "`rt findings`" + `
2. Verify findings are confirmed: ` + "`rt verify-finding <id> confirmed`" + `
3. Add recommendations: ` + "`rt recommend <id> \"Fix description\"`" + `
4. Check progress: ` + "`rt standup`" + `
5. Generate report:
   - Full: ` + "`rt report --html -o report.html`" + `
   - Executive: ` + "`rt report --html --exec -o exec.html`" + `
   - Technical: ` + "`rt report --html --tech -o tech.html`" + `
   - With branding: ` + "`rt report --html --template mycompany -o report.html`" + `
6. Export for tools:
   - ` + "`rt export ghostwriter -o findings.json`" + `
   - ` + "`rt export mitre -o navigator.json`" + `
`,
		"daily-standup.md": `# Daily Standup

Review progress and plan next steps.

1. Run standup: ` + "`rt standup`" + `
2. Check untested scope: ` + "`rt scope-untested`" + `
3. Review checklist: ` + "`rt checklist`" + `
4. Check session time: ` + "`rt time`" + `
5. Review recent evidence: ` + "`rt timeline`" + `
6. Summarize findings so far: ` + "`rt findings`" + `
`,
	}

	for name, content := range skills {
		if err := os.WriteFile(filepath.Join(skillsDir, name), []byte(content), 0600); err != nil {
			return err
		}
	}

	return nil
}
