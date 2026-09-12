package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
)

const rtBeginMarker = "<!-- RT:BEGIN -->"
const rtEndMarker = "<!-- RT:END -->"

var initWorkspaceCmd = &cobra.Command{
	Use:   "init-workspace",
	Short: "Add RT agent context to current directory (appends to CLAUDE.md + creates skills)",
	Long: `Appends RT agent context to CLAUDE.md (between RT:BEGIN/RT:END markers) and
creates .claude/commands/ skills. If CLAUDE.md already has RT markers, they are
replaced. If no CLAUDE.md exists, one is created. Existing content outside
the markers is preserved.

Run this in your pentest working directory before starting a session.
Use 'rt remove-workspace' to strip RT content from the workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()

		content := agentClaudeMD
		if remote.IsJoined() {
			state, err := remote.LoadState()
			if err == nil {
				content += remoteSection(state)
				fmt.Println("  Detected remote mode (joined server)")
			}
		}

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if err := appendOrReplaceRT(claudeMD, content); err != nil {
			return fmt.Errorf("update CLAUDE.md: %w", err)
		}
		fmt.Printf("  Updated: %s (RT section appended)\n", claudeMD)

		skillsDir := filepath.Join(dir, ".claude", "commands")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("create skills dir: %w", err)
		}

		skills := map[string]string{
			"pentest.md": skillPentest,
			"report.md":  skillReport,
		}

		for name, content := range skills {
			p := filepath.Join(skillsDir, name)
			if err := os.WriteFile(p, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
			fmt.Printf("  Created: %s\n", p)
		}

		fmt.Println()
		fmt.Println("  Workspace ready. An AI agent in this directory will:")
		fmt.Println("    1. Read CLAUDE.md -> understand RT discipline rules")
		fmt.Println("    2. Use /pentest <target> -> start engagement with evidence capture")
		fmt.Println("    3. Use /report -> verify findings + generate report")
		fmt.Println()
		fmt.Println("  Use 'rt remove-workspace' to strip RT content.")

		return nil
	},
}

var removeWorkspaceCmd = &cobra.Command{
	Use:   "remove-workspace",
	Short: "Remove RT agent context from current directory",
	Long:  `Strips the RT:BEGIN..RT:END section from CLAUDE.md and removes RT skill files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if err := stripRTSection(claudeMD); err != nil {
			fmt.Printf("  CLAUDE.md: %s\n", err)
		} else {
			fmt.Printf("  Cleaned: %s (RT section removed)\n", claudeMD)
		}

		skillsDir := filepath.Join(dir, ".claude", "commands")
		for _, name := range []string{"pentest.md", "report.md"} {
			p := filepath.Join(skillsDir, name)
			if err := os.Remove(p); err == nil {
				fmt.Printf("  Removed: %s\n", p)
			}
		}

		fmt.Println("  RT workspace content removed.")
		return nil
	},
}

func appendOrReplaceRT(path, rtContent string) error {
	block := "\n" + rtBeginMarker + "\n" + rtContent + "\n" + rtEndMarker + "\n"

	existing, err := os.ReadFile(path)
	if err != nil {
		return os.WriteFile(path, []byte(strings.TrimLeft(block, "\n")), 0644)
	}

	content := string(existing)
	beginIdx := strings.Index(content, rtBeginMarker)
	endIdx := strings.Index(content, rtEndMarker)

	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		content = content[:beginIdx] + strings.TrimLeft(block, "\n") + content[endIdx+len(rtEndMarker):]
		content = strings.TrimRight(content, "\n") + "\n"
		return os.WriteFile(path, []byte(content), 0644)
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	return os.WriteFile(path, []byte(content), 0644)
}

func stripRTSection(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("not found")
	}
	content := string(data)
	beginIdx := strings.Index(content, rtBeginMarker)
	endIdx := strings.Index(content, rtEndMarker)
	if beginIdx < 0 || endIdx < 0 {
		return fmt.Errorf("no RT section found")
	}

	before := strings.TrimRight(content[:beginIdx], "\n")
	after := strings.TrimLeft(content[endIdx+len(rtEndMarker):], "\n")
	result := before
	if after != "" {
		result += "\n\n" + after
	}
	result = strings.TrimSpace(result) + "\n"
	if strings.TrimSpace(result) == "" {
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(result), 0644)
}

const agentClaudeMD = `# RT — Red Team Evidence Discipline

You have ` + "`rt`" + ` (or ` + "`rt.exe`" + `) available — a red team evidence capture tool.
These rules ensure tamper-proof evidence collection during pentests.

## CRITICAL: Mode Detection

RT operates in two modes. **You MUST detect the correct mode BEFORE doing anything.**

**How to detect:**
- If the user provides a **server URL + API key** (e.g. ` + "`--server https://10.0.0.5:8443 --key rt_key_...`" + `, or says "remote to server X with key Y"), → use **REMOTE MODE**
- Otherwise → use **LOCAL MODE**

Look for these patterns in the user's request:
- Server URL: any ` + "`https://...`" + ` or ` + "`http://...`" + ` with a port (e.g. ` + "`:8443`" + `)
- API key: any string starting with ` + "`rt_key_`" + `
- Keywords: "remote", "server", "central", "join", "connect to", "sync to"

If you detect BOTH a server URL AND an API key → you are in **REMOTE MODE**.

---

## LOCAL MODE (default — no server URL / API key provided)

### Environment Setup
` + "```bash" + `
# Linux/Mac
export RT_HOME="$(pwd)/.rt"
# Windows PowerShell
$env:RT_HOME = "$PWD\.rt"
` + "```" + `

### Workflow
` + "```bash" + `
# Phase 1: Setup
rt new "<engagement-name>" --client "<client>"
rt unlock
rt scope <target-hosts>
rt checklist-load ptes
rt start
rt serve --listen localhost:7777 &

# Phase 2: Test — ALL commands through rt exec
rt exec -- <command>
rt finding "Title" --priority <level> --mitre <id>
rt cred <user> <secret> --host <ip>

# Phase 3: Wrap Up
rt findings
rt finding-note <id> "Step 1: ... Step 2: ... Result: ..."
rt verify-finding <id> confirmed
rt recommend <id> "Fix steps"
rt report --html -o report.html
rt stop && rt lock
` + "```" + `

---

## REMOTE MODE (server URL + API key provided)

**In remote mode, you do NOT create a local engagement.** All evidence goes directly to the central server. The dashboard is already running on the server — do NOT start ` + "`rt serve`" + ` locally.

### Setup (once per shell)
` + "```bash" + `
export RT_SERVER="<server-url>"       # e.g. https://10.0.0.5:8443
export RT_API_KEY="<api-key>"         # e.g. rt_key_xxx...
# For self-signed certs:
export RT_INSECURE=1
` + "```" + `

### Start a remote session
` + "```bash" + `
rt remote-session start --server $RT_SERVER --key $RT_API_KEY --operator $(whoami)
# Note the session ID printed (e.g. remote-a1b2c3d4)
` + "```" + `

### Test — ALL commands through rt remote-exec
` + "```bash" + `
rt remote-exec --server $RT_SERVER --key $RT_API_KEY --session <session-id> -- <command>
# Examples:
rt remote-exec --server $RT_SERVER --key $RT_API_KEY -- nmap -sV <target>
rt remote-exec --server $RT_SERVER --key $RT_API_KEY -- curl -s http://<target>/api
` + "```" + `

### Get engagement context from server
` + "```bash" + `
rt sync --server $RT_SERVER --key $RT_API_KEY
# Shows: scope hosts, existing findings, checklist progress
` + "```" + `

### Stop session when done
` + "```bash" + `
rt remote-session stop --session <session-id> --server $RT_SERVER --key $RT_API_KEY
` + "```" + `

### What you CANNOT do in remote mode
- ` + "`rt new`" + `, ` + "`rt unlock`" + `, ` + "`rt start`" + `, ` + "`rt stop`" + `, ` + "`rt lock`" + ` — the server manages the engagement
- ` + "`rt serve`" + ` — the dashboard is on the server, tell the user to open the server URL
- ` + "`rt finding`" + `, ` + "`rt cred`" + `, ` + "`rt report`" + ` — these need local DB; use the server dashboard instead
- ` + "`rt verify-chain`" + ` — chain is on the server

### What you CAN do in remote mode
- ` + "`rt remote-exec`" + ` — execute + submit evidence to server
- ` + "`rt remote-session`" + ` — start/stop sessions
- ` + "`rt sync`" + ` — pull context (scope, findings, checklist)

### Remote completion
Tell the user: "Evidence has been submitted to the central server at <server-url>. Open the dashboard there to review findings and generate the report."

---

## Evidence Rules (both modes)

1. **Every command against a target MUST go through RT** (` + "`rt exec`" + ` or ` + "`rt remote-exec`" + `)
2. **Record findings immediately** — in local mode: ` + "`rt finding`" + `, in remote mode: note findings in output, the lead reviews on the dashboard
3. **Record credentials immediately** — in local mode: ` + "`rt cred`" + `, in remote mode: they appear in evidence output and can be extracted on the server
4. **Use ` + "`--`" + ` separator** for flags: ` + "`rt exec -- curl -s http://target`" + `
5. **Screenshots are MANDATORY** for web vulnerabilities: ` + "`rt screenshot <file>`" + `
6. Every exploit step MUST be a captured command — milestones alone are NOT evidence

## Quick Reference

| Action | Local Mode | Remote Mode |
|--------|-----------|-------------|
| Execute + capture | ` + "`rt exec -- <cmd>`" + ` | ` + "`rt remote-exec --server $RT_SERVER --key $RT_API_KEY -- <cmd>`" + ` |
| Start session | ` + "`rt start`" + ` | ` + "`rt remote-session start --server ... --key ...`" + ` |
| Stop session | ` + "`rt stop`" + ` | ` + "`rt remote-session stop --session <id> ...`" + ` |
| Create finding | ` + "`rt finding \"Title\" --priority high`" + ` | Use server dashboard |
| Store credential | ` + "`rt cred <user> <secret>`" + ` | Auto-extracted from evidence on server |
| Sync context | N/A | ` + "`rt sync --server ... --key ...`" + ` |
| Dashboard | ` + "`rt serve --listen localhost:7777`" + ` | Already running on server |
| Generate report | ` + "`rt report --html`" + ` | Use server dashboard |`

const skillPentest = `# Pentest Target

You are a penetration tester. Use RT to capture ALL evidence.

## Input
$ARGUMENTS

## Step 0: Detect Mode (DO THIS FIRST)

Parse the input above. Look for:
- A **server URL** (` + "`https://...`" + ` or ` + "`http://...`" + ` with a port like ` + "`:8443`" + `)
- An **API key** (starts with ` + "`rt_key_`" + `)
- Keywords like "remote", "server", "connect", "join"

**If you find BOTH a server URL AND an API key → go to REMOTE WORKFLOW below.**
**Otherwise → go to LOCAL WORKFLOW below.**

---

## LOCAL WORKFLOW (no server/key provided)

### Step 1: Environment
` + "```bash" + `
# Windows PowerShell:
$env:RT_HOME = "$PWD\.rt"
# Linux/Mac:
export RT_HOME="$(pwd)/.rt"
` + "```" + `

### Step 2: Create Engagement + Start Dashboard
` + "```bash" + `
rt new "<name-from-target>" --client "Pentest"
rt unlock
rt scope <target-host>
rt checklist-load ptes
rt start
rt serve --listen localhost:7777 &
` + "```" + `
**Tell the user**: "Dashboard is live at http://localhost:7777 — open it for real-time tracking."

### Step 3: Test (ALL commands through rt exec)
- ` + "`rt exec -- <cmd>`" + ` — every target command (use ` + "`--`" + ` before flags)
- ` + "`rt finding \"Title\" --priority <sev> --mitre <id>`" + ` — every vuln found
- ` + "`rt cred <user> <secret> --host <ip>`" + ` — every credential found
- ` + "`rt milestone \"Description\"`" + ` — every major achievement
- ` + "`rt screenshot <file>`" + ` — MANDATORY for web vulns

### Step 4: Wrap Up (MANDATORY)
` + "```bash" + `
rt findings
rt finding-note <id> "Step 1: <cmd> Step 2: <observation> Result: <what was achieved>"
rt verify-finding <id> confirmed --screenshot <file>
rt recommend <id> "Fix description"
rt standup && rt verify-chain
rt report --html -o report.html
rt stop && rt lock
` + "```" + `

**STOP GATE — verify ALL before presenting results:**
1. ` + "`rt serve`" + ` is running, user told to open dashboard
2. Every finding has PoC notes, verification, and recommendations
3. ` + "`rt report --html`" + ` generated, ` + "`rt verify-chain`" + ` passed

---

## REMOTE WORKFLOW (server URL + API key detected)

**You do NOT create a local engagement. Evidence goes directly to the central server.**

### Step 1: Set environment variables
` + "```bash" + `
export RT_SERVER="<server-url>"       # from the user's input
export RT_API_KEY="<api-key>"         # from the user's input
# For self-signed certs (common in pentest labs):
export RT_INSECURE=1
` + "```" + `

### Step 2: Start remote session
` + "```bash" + `
rt remote-session start --server $RT_SERVER --key $RT_API_KEY --operator $(whoami) --insecure
# Save the session ID that is printed!
` + "```" + `

### Step 3: Get context from server
` + "```bash" + `
rt sync --server $RT_SERVER --key $RT_API_KEY --insecure
# Shows scope, existing findings, checklist — understand what is already done
` + "```" + `

### Step 4: Test (ALL commands through rt remote-exec)
` + "```bash" + `
# Every command against the target:
rt remote-exec --server $RT_SERVER --key $RT_API_KEY --session <session-id> --insecure -- <command>

# Examples:
rt remote-exec --server $RT_SERVER --key $RT_API_KEY --session <session-id> --insecure -- nmap -sV <target>
rt remote-exec --server $RT_SERVER --key $RT_API_KEY --session <session-id> --insecure -- curl -s http://<target>/api
rt remote-exec --server $RT_SERVER --key $RT_API_KEY --session <session-id> --insecure -- gobuster dir -u http://<target> -w /usr/share/wordlists/common.txt
` + "```" + `

### Step 5: Wrap up
` + "```bash" + `
rt remote-session stop --session <session-id> --server $RT_SERVER --key $RT_API_KEY --insecure
` + "```" + `
**Tell the user**: "All evidence has been submitted to the central server at <server-url>. Open the dashboard there to review findings, manage credentials, and generate the report."

### Remote mode rules
- Do NOT run ` + "`rt new`" + `, ` + "`rt unlock`" + `, ` + "`rt start`" + `, ` + "`rt serve`" + `, ` + "`rt lock`" + ` — the server handles all of this
- Do NOT run ` + "`rt exec`" + ` — use ` + "`rt remote-exec`" + ` instead
- Findings and credentials are visible on the server dashboard — the lead manages them there
- Use ` + "`rt sync`" + ` to check what scope/findings/checklist already exist

---

## Evidence Quality Rules (BOTH modes)

1. **Every exploit step MUST be a captured command** — milestones alone are NOT evidence
2. **If a command fails, diagnose and re-run correctly** (on Windows: ` + "`curl.exe`" + ` not ` + "`curl`" + `)
3. **Store ALL secrets** (passwords, JWT secrets, tokens, API keys) — in local mode as ` + "`rt cred`" + `, in remote mode they appear in evidence output
4. **Screenshots are MANDATORY** for web vulnerabilities
5. **Use ` + "`--`" + ` separator** for command flags: ` + "`... -- curl -s -X POST ...`" + `

You decide the attack strategy. RT is your evidence pipeline, not your playbook.
Think like a pentester: enumerate, analyze, exploit, escalate, document.
`

func remoteSection(state *remote.ConnectionState) string {
	var b strings.Builder
	b.WriteString("\n\n## Remote Mode (Connected to Central Server)\n\n")
	b.WriteString("This workspace is joined to a central RT server. Use remote commands instead of local ones.\n\n")
	b.WriteString("**Server:** `" + state.ServerURL + "`\n")
	b.WriteString("**Operator:** `" + state.Operator + "`\n\n")
	b.WriteString("### Remote Commands\n\n")
	b.WriteString("Instead of `rt exec`, use:\n")
	b.WriteString("```bash\n")
	b.WriteString("rt exec --server " + state.ServerURL + " --key <YOUR_KEY> -- <command>\n")
	b.WriteString("```\n\n")
	b.WriteString("Or set environment variables (recommended):\n")
	b.WriteString("```bash\n")
	b.WriteString("# Set once per shell\n")
	if state.Insecure {
		b.WriteString("export RT_SERVER=" + state.ServerURL + "\n")
	} else {
		b.WriteString("export RT_SERVER=" + state.ServerURL + "\n")
	}
	b.WriteString("export RT_API_KEY=<your-api-key>\n")
	if state.Insecure {
		b.WriteString("export RT_INSECURE=1\n")
	}
	b.WriteString("\n# Then just use:\n")
	b.WriteString("rt exec -- <command>\n")
	b.WriteString("```\n\n")
	b.WriteString("### Sync Context\n\n")
	b.WriteString("Pull engagement context (scope, checklist, findings) from the server:\n")
	b.WriteString("```bash\n")
	b.WriteString("rt sync\n")
	b.WriteString("```\n\n")
	b.WriteString("### Key Differences from Local Mode\n\n")
	b.WriteString("- Evidence is submitted to the central server, not stored locally\n")
	b.WriteString("- The dashboard runs on the server — do NOT start `rt serve` locally\n")
	b.WriteString("- Sessions are managed server-side (`rt join` / `rt leave`)\n")
	b.WriteString("- All operators see evidence in real-time via the server dashboard\n")
	return b.String()
}

const skillReport = `# Generate Pentest Report

Verify all findings, add recommendations, and generate the final report.

## Steps

1. Review all findings:
` + "```bash" + `
./rt findings
` + "```" + `

2. For each unverified finding, verify it:
` + "```bash" + `
./rt verify-finding <id> confirmed --note "Verification details"
# Add --screenshot <file> for visual evidence on target
` + "```" + `

3. For each finding without a recommendation:
` + "```bash" + `
./rt recommend <id> "Specific remediation steps"
` + "```" + `

4. Final checks:
` + "```bash" + `
./rt standup
./rt verify-chain
./rt scope-untested
` + "```" + `

5. Generate reports:
` + "```bash" + `
./rt report --html -o report.html
./rt report -o report.md
./rt export json -o evidence.json
` + "```" + `

6. Present summary: findings by severity, scope coverage, time spent.
`
