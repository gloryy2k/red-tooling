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
rt finding "Title" --priority critical --mitre T1190 --host 10.0.0.1
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

### Remote Mode (central server)
` + "```bash" + `
# Set env vars once — all rt commands read them automatically:
export RT_SERVER="https://server:8443"
export RT_API_KEY="rt_key_..."
export RT_INSECURE=1                   # for self-signed certs

rt remote-session start                # start remote session
rt remote-exec --session <id> -- <cmd> # exec + submit evidence
rt remote-session stop --session <id>  # stop session
rt sync                                # pull scope/findings/checklist from server
` + "```" + `

### Dashboard
` + "```bash" + `
rt serve --listen localhost:8080       # start web dashboard
` + "```" + `

## Workflow for Agents

1. **Always capture evidence** — use ` + "`rt exec <cmd>`" + ` instead of raw commands
2. **Auto-flag** handles tagging — RT detects credentials, admin access, vulns, etc.
3. **Create findings for ALL security-significant discoveries** — not just exploitable vulns (see below)
4. **Mark scope** as tested after scanning each host
5. **Store credentials** immediately when found — ` + "`rt cred <user> <secret> --host <ip>`" + `
6. **Check progress** with ` + "`rt standup`" + ` and ` + "`rt scope-untested`" + `
7. **Generate report** when phase is complete

## When to Create Findings (IMPORTANT)

Create a finding for every discovery with security significance. Findings populate the ATT&CK map on the dashboard — if you skip creating a finding, the discovery is invisible on the kill chain.

**ALWAYS create a finding for:**
- Exploitable vulnerabilities (critical/high) — T1190, T1068, T1548, etc.
- Credential discoveries — dumped hashes (T1003), kerberoast (T1558), cleartext creds (T1552)
- Privilege escalation paths — misconfigs, unquoted services (T1574), token impersonation (T1134)
- Significant recon discoveries — domain admin enumeration (T1087), network share access (T1135), trust relationships
- Lateral movement success — pass-the-hash (T1550), RDP/SMB access (T1021)
- Persistence mechanisms found — scheduled tasks (T1053), services (T1543), registry keys (T1547)
- Defense evasion — AV disabled (T1562), AMSI bypass, logging gaps (T1070)
- Misconfigurations with security impact — weak ACLs, GPO abuse paths, certificate template vulns

**DO NOT create a finding for:**
- Basic connectivity checks (ping, curl health checks)
- Normal enumeration with no security-relevant result (e.g. ` + "`nmap`" + ` that only finds expected open ports)
- Failed exploitation attempts (log as evidence only)
- Standard tool output with no actionable information

**Priority guide:**
- **critical** — direct path to domain admin / full compromise
- **high** — exploitable vuln, credential access, privilege escalation
- **medium** — significant misconfiguration, information disclosure with impact
- **low** — informational finding with security relevance (weak policy, missing hardening)
- **info** — notable observation for the report (architecture notes, attack surface mapping)

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
3. Run scans: ` + "`rt exec nmap -sV -sC <target>`" + `
4. Import results: ` + "`rt import {work_dir}/nmap.xml`" + `
5. Mark hosts tested: ` + "`rt scope-tested <ip>`" + `
6. **Create findings for significant discoveries:**
   - Interesting services found → ` + "`rt finding \"SMB signing disabled on DC\" --priority medium --mitre T1557`" + `
   - User/group enumeration → ` + "`rt finding \"Domain Admins enumerated\" --priority low --mitre T1087`" + `
   - Network shares accessible → ` + "`rt finding \"Writable network share found\" --priority medium --mitre T1135`" + `
   - Do NOT create findings for routine port scans with no security-relevant result
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
