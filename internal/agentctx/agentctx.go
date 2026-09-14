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

	b.WriteString("## RT — Evidence Logger\n\n")
	b.WriteString("RT captures every command you run as evidence, syncs to a central dashboard, and maps findings to ATT&CK. You are the operator — RT is your logger.\n\n")

	b.WriteString("## First Run (do this once)\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# 1. Connect to the central server\n")
	b.WriteString("rt join --server https://SERVER:PORT --key API_KEY --insecure\n\n")
	b.WriteString("# 2. Verify connection\n")
	b.WriteString("rt status\n\n")
	b.WriteString("# 3. Add scope targets\n")
	b.WriteString("rt scope TARGET_HOSTS\n")
	b.WriteString("```\n\n")
	b.WriteString("After `rt join`, RT persists connection state to `~/.rt/remote.json` and server config to `~/.rt/server.json`.\n")
	b.WriteString("**Do NOT export RT_SERVER, RT_API_KEY, or RT_INSECURE as environment variables. Ever.**\n")
	b.WriteString("From this point on, just run `rt exec` directly — it auto-syncs to the server.\n\n")
	b.WriteString("If you disconnect and need to rejoin later: `rt join` (no flags — RT remembers).\n\n")

	b.WriteString("## Capturing Evidence\n\n")
	b.WriteString("**Every command MUST go through `rt exec`:**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("rt exec nmap -sV -sC 10.0.0.1\n")
	b.WriteString("rt exec --cmd \"netexec smb 10.0.0.1 -u '' -p '' --shares\"\n")
	b.WriteString("```\n\n")
	b.WriteString("Use `--cmd` when the command has quotes, pipes, or special characters.\n\n")

	b.WriteString("### Output Management\n\n")
	b.WriteString("- **Brute-force / password spray** — always use wordlist files, never loop individual commands\n")
	b.WriteString("  - Bad: 16 separate `rt exec curl ... -d '{\"password\":\"X\"}'`\n")
	b.WriteString("  - Good: `rt exec --cmd \"netexec smb target -u user -p /tmp/wordlist.txt\"`\n")
	b.WriteString("- **Output > ~50 lines expected** — pipe: `rt exec --cmd \"cmd | head -100\"` or `| grep -i keyword`\n")
	b.WriteString("- **Binary/JS/large file downloads** — don't capture via rt exec. Download normally, then `rt note \"Downloaded X to /tmp/X\"`\n")
	b.WriteString("- **Noisy tools (hydra, gobuster, ffuf)** — filter output: `| grep -E 'found|SUCCESS|\\[\\+\\]'`\n")
	b.WriteString("- **Nmap XML output** — save to file then import: `rt exec --cmd \"nmap -oX /tmp/scan.xml ...\"` then `rt import /tmp/scan.xml`\n\n")

	b.WriteString("### Offline Resilience\n\n")
	b.WriteString("If the server is unreachable, `rt exec` still runs the command and queues evidence to `~/.rt/.sync-queue.json`. When connection restores, queued items auto-drain on the next `rt exec`. You will see:\n")
	b.WriteString("```\n[sync] Server unreachable — evidence queued (N pending)\n```\n")
	b.WriteString("**This is normal. Keep working.** Evidence will sync when the server comes back. Check queue size with `rt status`.\n\n")

	b.WriteString("## Creating Findings\n\n")
	b.WriteString("Findings populate the ATT&CK kill chain on the dashboard. **No finding = invisible on the map.**\n\n")
	b.WriteString("```bash\nrt finding \"Title\" --priority high --mitre T1078 --host 10.0.0.1\n```\n\n")

	b.WriteString("### The Rule\n\n")
	b.WriteString("After each significant command, ask: **\"Did I just discover access, credentials, a vulnerability, or a misconfiguration with security impact?\"** If yes → create a finding. If unsure → create it (unverified findings cost nothing; missing findings leave gaps on the kill chain).\n\n")
	b.WriteString("Before creating, quick-check for duplicates: `rt findings` to see what already exists.\n\n")

	b.WriteString("### Finding Categories\n\n")

	b.WriteString("**1. Access Gained** — you can now reach or control something you shouldn't\n\n")
	b.WriteString("Priority: critical if admin/root, high if regular user, medium if limited/read-only\n\n")
	b.WriteString("MITRE: T1078 (valid accounts), T1021 (remote services), T1133 (external remote services)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- SMB/WinRM/RDP login succeeds → \"Valid Credentials: {user} on {host} via {protocol}\"\n")
	b.WriteString("- SSH access to Linux host → \"SSH Access: {user} on {host}\"\n")
	b.WriteString("- Web admin panel login → \"Admin Access to {service} on {host}:{port}\"\n")
	b.WriteString("- API authentication with found token → \"API Access to {service} on {host}\"\n")
	b.WriteString("- Database login → \"Database Access: {user} on {host} ({db_type})\"\n\n")

	b.WriteString("**2. Credentials Discovered** — passwords, hashes, tokens, keys\n\n")
	b.WriteString("Priority: critical if from config/file extraction, high if cracked/brute-forced, medium if default\n\n")
	b.WriteString("MITRE: T1552 (unsecured creds), T1003 (credential dumping), T1110 (brute force), T1558 (kerberos)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- Password in config/policy/source code → \"Credentials in {source} on {host}\"\n")
	b.WriteString("- Brute-force/spray success → \"Weak Password: {user} on {host}\"\n")
	b.WriteString("- NTLM hash dumped → \"NTLM Hash Extracted: {user} on {host}\"\n")
	b.WriteString("- Kerberoast/AS-REP ticket → \"Kerberoastable Account: {user}\"\n")
	b.WriteString("- API key/token in exposed file → \"API Key Exposed in {location}\"\n")
	b.WriteString("- Database creds in connection string → \"Database Credentials in {source}\"\n\n")

	b.WriteString("**3. Vulnerability / Exploit** — something technically exploitable\n\n")
	b.WriteString("Priority: critical if RCE, high if auth bypass or privesc, medium if limited impact\n\n")
	b.WriteString("MITRE: T1190 (exploit public-facing), T1068 (privesc), T1210 (exploit remote services)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- SQL injection → \"SQL Injection in {param} on {host}\"\n")
	b.WriteString("- RCE via any method → \"Remote Code Execution via {method} on {host}\"\n")
	b.WriteString("- Privilege escalation → \"Privilege Escalation via {method} on {host}\"\n")
	b.WriteString("- Known CVE → \"{CVE} Exploitable on {host}\"\n\n")

	b.WriteString("**4. Misconfiguration** — security feature missing or wrong\n\n")
	b.WriteString("Priority: medium typically, high if directly exploitable\n\n")
	b.WriteString("MITRE: T1557 (MitM/signing), T1135 (network shares), T1562 (impair defenses)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- Protocol signing disabled → \"{protocol} Signing Not Required on {host}\"\n")
	b.WriteString("- Anonymous/guest write access → \"Guest Write Access to {share} on {host}\"\n")
	b.WriteString("- Directory listing on web → \"Directory Listing Enabled on {host}:{port}\"\n")
	b.WriteString("- Anonymous bind (FTP/LDAP/SNMP) → \"Anonymous {protocol} Access on {host}\"\n")
	b.WriteString("- Weak TLS/SSL → \"Weak TLS Configuration on {host}:{port}\"\n")
	b.WriteString("- Default credentials on any service → \"Default Credentials on {service} ({user})\"\n")
	b.WriteString("- No brute-force protection → \"No Rate-Limiting on {service}\"\n\n")

	b.WriteString("**5. Enumeration with Security Impact** — learned something an attacker values\n\n")
	b.WriteString("Priority: medium for user/group enum, low for architecture observations\n\n")
	b.WriteString("MITRE: T1087 (account discovery), T1046 (service scanning), T1018 (remote system discovery)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- RID brute-force / LDAP enum reveals users → \"User Enumeration via {method} on {host}\"\n")
	b.WriteString("- SNMP default community string → \"SNMP Default Community on {host}\"\n")
	b.WriteString("- DNS zone transfer → \"DNS Zone Transfer on {host}\"\n")
	b.WriteString("- Cloud metadata accessible → \"Cloud Metadata Exposed on {host}\"\n\n")

	b.WriteString("**6. Lateral Movement / Pivot** — moved to a new host or network segment\n\n")
	b.WriteString("Priority: high\n\n")
	b.WriteString("MITRE: T1550 (alternate auth), T1021 (remote services), T1563 (session hijacking)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- Pass-the-hash/pass-the-ticket → \"Pass-the-Hash: {user} on {host}\"\n")
	b.WriteString("- Pivoted to new host → \"Lateral Movement via {method} to {host}\"\n")
	b.WriteString("- Session hijacking → \"Session Hijack on {host}\"\n\n")

	b.WriteString("**7. Persistence / Defense Evasion** — established persistence or bypassed controls\n\n")
	b.WriteString("Priority: high\n\n")
	b.WriteString("MITRE: T1053 (scheduled tasks), T1543 (services), T1547 (autostart), T1562 (impair defenses)\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("- Cron/scheduled task → \"Persistence via {mechanism} on {host}\"\n")
	b.WriteString("- AV/EDR disabled or bypassed → \"Security Controls Disabled on {host}\"\n")
	b.WriteString("- Service/config tampered → \"{service} Configuration Tampered on {host}\"\n")
	b.WriteString("- Log cleared/tampered → \"Audit Log Cleared on {host}\"\n\n")

	b.WriteString("### Do NOT create findings for\n\n")
	b.WriteString("- Basic connectivity (ping, traceroute)\n")
	b.WriteString("- Port scans showing only expected services\n")
	b.WriteString("- Failed exploitation attempts (evidence is enough)\n")
	b.WriteString("- Unauthenticated requests that return a login page\n")
	b.WriteString("- Normal tool output with no security insight\n\n")

	b.WriteString("### Priority Quick Reference\n\n")
	b.WriteString("| Priority | Meaning |\n")
	b.WriteString("|----------|---------|")
	b.WriteString("\n| critical | Domain admin / full compromise / creds from config / RCE |")
	b.WriteString("\n| high | Valid creds, exploitable vuln, privesc, lateral movement |")
	b.WriteString("\n| medium | Misconfiguration, user enum, exposed service, info disclosure |")
	b.WriteString("\n| low | Weak policy, missing hardening, architecture observation |\n\n")

	b.WriteString("## Finding Lifecycle\n\n")
	b.WriteString("Creating a finding is step 1. Complete the lifecycle:\n\n")
	b.WriteString("**1. Create** — immediately when discovered\n")
	b.WriteString("```bash\nrt finding \"SMB Signing Disabled on 10.0.0.1\" --priority medium --mitre T1557.001 --host 10.0.0.1\n```\n\n")
	b.WriteString("**2. Verify** — when you have confirmed proof\n")
	b.WriteString("```bash\n")
	b.WriteString("rt verify-finding <id> confirmed       # you proved it's real and exploitable\n")
	b.WriteString("rt verify-finding <id> false-positive   # turned out to be a false alarm\n")
	b.WriteString("```\n")
	b.WriteString("Verify as `confirmed` when you have evidence that directly demonstrates the issue (successful login, extracted data, executed command). Leave as `unverified` if you suspect but haven't proven it.\n\n")
	b.WriteString("**3. Recommend** — add remediation for confirmed findings\n")
	b.WriteString("```bash\nrt recommend <id> \"Disable guest access to SMB shares and enable SMB signing via Group Policy\"\n```\n")
	b.WriteString("Keep recommendations actionable and specific — tell the client WHAT to do, not just \"fix this\".\n\n")

	b.WriteString("## Storing Credentials\n\n")
	b.WriteString("```bash\nrt cred <username> <secret> --host <ip>\n```\n\n")
	b.WriteString("- Store **immediately** when discovered — don't wait\n")
	b.WriteString("- `--host` = the IP where the credential was **validated or extracted from** (always a target IP, never your machine)\n")
	b.WriteString("- Store all types: plaintext passwords, NTLM hashes, Kerberos tickets, API tokens, SSH keys, private keys\n")
	b.WriteString("- If a cred works on multiple hosts, store once with the host where you first confirmed it\n\n")

	b.WriteString("## Scope Management\n\n")
	b.WriteString("### Initial scope\n")
	b.WriteString("```bash\nrt scope 10.0.0.1,10.0.0.2,10.0.0.0/24\n```\n\n")
	b.WriteString("### Discovered hosts\n")
	b.WriteString("When you discover new hosts during the engagement (DNS records, ARP, pivot, subnet scan):\n")
	b.WriteString("- **Inside authorized scope** → `rt scope <new-host>` to add, then test\n")
	b.WriteString("- **Outside authorized scope** → `rt note \"Discovered {host} — outside scope, not testing\"` — do NOT add or test\n\n")
	b.WriteString("### Mark tested\n")
	b.WriteString("```bash\nrt scope-tested <ip>\n```\n")
	b.WriteString("Run this after you have done meaningful testing on a host (not just a single ping). Ideally before moving to the next target.\n\n")
	b.WriteString("### Check progress\n")
	b.WriteString("```bash\n")
	b.WriteString("rt scope-list          # all hosts + tested status\n")
	b.WriteString("rt scope-untested      # what's left\n")
	b.WriteString("```\n\n")

	b.WriteString("## Collaboration & Sync\n\n")
	b.WriteString("Other operators or the lead may be working in parallel — adding scope, creating findings, storing credentials.\n\n")
	b.WriteString("```bash\nrt sync                # pull latest scope, findings, checklist from server\n```\n\n")
	b.WriteString("**When to sync:**\n")
	b.WriteString("- At the start of your session (after `rt join`)\n")
	b.WriteString("- Before starting work on a new target (check if someone else already tested it)\n")
	b.WriteString("- When you're unsure whether a finding already exists\n\n")
	b.WriteString("**Avoid duplicate work:** Run `rt findings` before creating a finding to check if someone else already logged the same discovery.\n\n")

	b.WriteString("## Import Tool Output\n\n")
	b.WriteString("When a tool produces structured output files, import them:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("rt import /tmp/scan.xml          # nmap XML\n")
	b.WriteString("rt import /tmp/results.json      # nuclei JSON\n")
	b.WriteString("rt import /tmp/data.csv          # generic CSV\n")
	b.WriteString("```\n\n")
	b.WriteString("Import after the scan completes — the raw `rt exec` captures the run, and `rt import` enriches the data.\n\n")

	b.WriteString("## Progress Tracking\n\n")
	b.WriteString("**After each significant discovery:**\n")
	b.WriteString("```bash\n")
	b.WriteString("rt finding \"...\" --priority ... --mitre ... --host ...   # log the finding\n")
	b.WriteString("rt cred ...                                               # store any credentials\n")
	b.WriteString("rt scope-tested <ip>                                      # if you've tested the host\n")
	b.WriteString("```\n\n")
	b.WriteString("**Every ~15-20 commands:**\n")
	b.WriteString("```bash\nrt status              # verify connection + check sync queue\n```\n\n")
	b.WriteString("**When switching targets or wrapping up:**\n")
	b.WriteString("```bash\n")
	b.WriteString("rt scope-tested <ip>   # mark current target done\n")
	b.WriteString("rt findings            # review what you've logged\n")
	b.WriteString("rt standup             # progress summary\n")
	b.WriteString("```\n\n")

	b.WriteString("## Notes\n\n")
	b.WriteString("Record observations that aren't findings or evidence:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("rt note \"Target is a Nessus scanner server, Windows Server 2022 Build 20348\"\n")
	b.WriteString("rt note \"SMB share 'Documents' contains scan reports — may have creds in policies\"\n")
	b.WriteString("rt note \"Host went offline at 08:12, came back at 08:14\"\n")
	b.WriteString("rt note \"Discovered internal subnet 10.10.10.0/24 via ARP — outside scope\"\n")
	b.WriteString("```\n\n")
	b.WriteString("Use notes for: architecture observations, reconnaissance context, timeline events, out-of-scope discoveries, attack planning rationale.\n\n")

	b.WriteString("## Command Reference\n\n")
	b.WriteString("| Command | Purpose |\n")
	b.WriteString("|---------|---------|")
	b.WriteString("\n| `rt join --server URL --key KEY --insecure` | Connect (first time) |")
	b.WriteString("\n| `rt join` | Reconnect (remembers server) |")
	b.WriteString("\n| `rt leave` | Disconnect |")
	b.WriteString("\n| `rt status` | Connection + sync queue |")
	b.WriteString("\n| `rt exec <cmd>` | Run + capture evidence |")
	b.WriteString("\n| `rt exec --cmd \"complex cmd\"` | Run with special chars |")
	b.WriteString("\n| `rt finding \"Title\" --priority P --mitre Txxxx --host IP` | Create finding |")
	b.WriteString("\n| `rt verify-finding <id> confirmed` | Verify finding |")
	b.WriteString("\n| `rt recommend <id> \"Fix description\"` | Add remediation |")
	b.WriteString("\n| `rt cred USER SECRET --host IP` | Store credential |")
	b.WriteString("\n| `rt note \"text\"` | Add observation |")
	b.WriteString("\n| `rt scope HOST1,HOST2` | Add to scope |")
	b.WriteString("\n| `rt scope-tested HOST` | Mark host tested |")
	b.WriteString("\n| `rt scope-list` | View scope + status |")
	b.WriteString("\n| `rt scope-untested` | List untested hosts |")
	b.WriteString("\n| `rt findings` | List all findings |")
	b.WriteString("\n| `rt creds` | List stored credentials |")
	b.WriteString("\n| `rt standup` | Progress summary |")
	b.WriteString("\n| `rt sync` | Pull context from server |")
	b.WriteString("\n| `rt import FILE` | Import nmap/nuclei/CSV |")
	b.WriteString("\n| `rt search \"keyword\"` | Search evidence |\n")

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
3. Run scans: ` + "`rt exec --cmd \"nmap -sV -sC -oX /tmp/scan.xml <target>\"`" + `
4. Import results: ` + "`rt import /tmp/scan.xml`" + `
5. Mark hosts tested: ` + "`rt scope-tested <ip>`" + `
6. Create findings for significant discoveries (see CLAUDE.md Finding Categories)
7. Store any credentials found: ` + "`rt cred <user> <pass> --host <ip>`" + `
`,
		"exploit.md": `# Exploit Target

Run exploitation against identified vulnerabilities.

1. Review findings: ` + "`rt findings`" + `
2. For each vuln, use rt exec to capture the exploit:
   ` + "`rt exec <exploit-command>`" + `
3. On success:
   - Create finding: ` + "`rt finding \"Title\" --priority critical --mitre T1190 --host <ip>`" + `
   - Store creds: ` + "`rt cred <user> <pass> --host <ip>`" + `
   - Verify finding: ` + "`rt verify-finding <id> confirmed`" + `
   - Add remediation: ` + "`rt recommend <id> \"Remediation steps\"`" + `
4. Mark scope: ` + "`rt scope-tested <ip>`" + `
`,
		"post-exploit.md": `# Post-Exploitation

Run post-exploitation on compromised hosts.

1. Enumerate the host:
   ` + "`rt exec --cmd \"whoami /all\"`" + ` or ` + "`rt exec id`" + `
2. Check for privilege escalation paths
3. Dump credentials if possible
4. Look for lateral movement opportunities
5. Create findings for each escalation path found
6. Store all credentials: ` + "`rt cred <user> <pass> --host <ip>`" + `
7. Update scope: ` + "`rt scope-tested <ip>`" + `
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
6. Export for tools:
   - ` + "`rt export ghostwriter -o findings.json`" + `
   - ` + "`rt export mitre -o navigator.json`" + `
`,
		"standup.md": `# Daily Standup

Review progress and plan next steps.

1. Run standup: ` + "`rt standup`" + `
2. Check untested scope: ` + "`rt scope-untested`" + `
3. Review recent evidence: ` + "`rt timeline`" + `
4. Summarize findings: ` + "`rt findings`" + `
5. Check connection: ` + "`rt status`" + `
6. Sync from server: ` + "`rt sync`" + `
`,
	}

	for name, content := range skills {
		if err := os.WriteFile(filepath.Join(skillsDir, name), []byte(content), 0600); err != nil {
			return err
		}
	}

	return nil
}
