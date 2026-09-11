package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if err := appendOrReplaceRT(claudeMD, agentClaudeMD); err != nil {
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

## CRITICAL: Environment Setup

**BEFORE any RT command**, you MUST set RT_HOME to the current project directory:
` + "```bash" + `
# Linux/Mac
export RT_HOME="$(pwd)/.rt"

# Windows PowerShell
$env:RT_HOME = "$PWD\.rt"

# Windows CMD
set RT_HOME=%cd%\.rt
` + "```" + `

**Every shell command that calls ` + "`rt`" + ` must have RT_HOME set.** If you open a new shell or use a different tool, re-set it. Data goes to the WRONG location without this.

## Rules

1. **Every command against a target MUST go through ` + "`rt exec`" + `.**
   ` + "```" + `
   # WRONG: evidence lost
   curl http://target/api/users

   # RIGHT: captured, auto-flagged, hash-chained
   rt exec curl http://target/api/users
   ` + "```" + `

2. **Record findings immediately.** When you discover a vulnerability:
   ` + "`rt finding \"Title\" --priority <level> --mitre <id>`" + `

3. **Record credentials immediately.** When you find creds:
   ` + "`rt cred <user> <secret> --host <ip>`" + `

4. **Mark milestones.** For major achievements (shell, admin, flag):
   ` + "`rt milestone \"Description\"`" + `

5. **Track scope.** Mark hosts tested: ` + "`rt scope-tested <host>`" + `

6. **Verify findings.** Before reporting, verify each finding:
   ` + "`rt verify-finding <id> confirmed --screenshot <file>`" + ` (visual evidence)
   ` + "`rt verify-finding <id> confirmed --no-screenshot`" + ` (non-visual)

7. **Add recommendations.** Every finding needs remediation guidance:
   ` + "`rt recommend <id> \"Specific fix\"`" + `

## Mandatory Workflow

### Phase 1: Setup + Dashboard (BEFORE testing)
` + "```bash" + `
rt new "<engagement-name>" --client "<client>"
rt unlock
rt scope <target-hosts>
rt checklist-load ptes
rt start
rt serve --listen localhost:7777 &
# Tell the user to open http://localhost:7777 for live tracking
` + "```" + `

### Phase 2: Testing
Use ` + "`rt exec`" + ` for ALL target commands. Record findings/creds/milestones as you go.
The dashboard at http://localhost:7777 updates in real-time via WebSocket.

**Evidence Quality Rules (CRITICAL):**
- Every exploit step MUST be an ` + "`rt exec`" + ` command — milestones alone are NOT evidence
- If a command fails, diagnose and re-run correctly (on Windows: use ` + "`curl.exe`" + ` not ` + "`curl`" + `)
- Use ` + "`--`" + ` separator for flags: ` + "`rt exec -- curl.exe -s http://target`" + `
- Store ALL secrets as credentials: passwords, JWT secrets, cookies/tokens, API keys
  - Example: ` + "`rt cred admin_jwt <token-value> --host <ip>`" + `
- Write PoC steps in finding notes: ` + "`rt finding-note <id> \"Step 1: ... Step 2: ...\"`" + `
  - Or use ` + "`--note`" + ` flag: ` + "`rt finding \"Title\" --priority critical --note \"PoC: ...\"`" + `
- Screenshots are MANDATORY for web vulnerabilities: ` + "`rt screenshot <file>`" + `

### Phase 3: Wrap Up (MANDATORY — do not skip)
` + "```bash" + `
# Write PoC notes for EVERY finding (report uses these)
rt findings                          # list all — note the IDs
rt finding-note <id> "Step 1: <exact command>
Step 2: <observation>
Result: <what was achieved>"         # repeat for EACH finding

rt verify-finding <id> confirmed     # verify each
rt recommend <id> "Fix steps"        # recommend each
rt standup                           # summary of what was done
rt verify-chain                      # verify evidence integrity
rt report --html -o report.html      # generate report
rt stop                              # stop session
rt lock                              # re-encrypt database
` + "```" + `

## Quick Reference

| Action | Command |
|--------|---------|
| Execute + capture | ` + "`rt exec -- <cmd>`" + ` |
| Create finding | ` + "`rt finding \"Title\" --priority high --mitre T1190`" + ` |
| Add PoC notes | ` + "`rt finding-note <id> \"Step 1: ... Result: ...\"`" + ` |
| Store credential | ` + "`rt cred <user> <secret> --host <ip>`" + ` |
| Mark milestone | ` + "`rt milestone \"Got admin\"`" + ` |
| Attach screenshot | ` + "`rt screenshot <file> [evidence-id]`" + ` |
| Verify finding | ` + "`rt verify-finding <id> confirmed`" + ` |
| Add recommendation | ` + "`rt recommend <id> \"Fix\"`" + ` |
| Check notes exist | ` + "`rt query \"SELECT id,title,notes FROM findings\"`" + ` |
| Dashboard | ` + "`rt serve --listen localhost:7777`" + ` |
| Generate report | ` + "`rt report --html -o report.html`" + ` |

## What NOT to do

- NEVER run rt commands without setting RT_HOME first
- NEVER run target commands outside ` + "`rt exec`" + `
- NEVER leave findings unverified or without recommendations
- NEVER skip ` + "`rt cred`" + ` when credentials are found (passwords, tokens, cookies, API keys — ALL of them)
- NEVER skip ` + "`rt milestone`" + ` for major achievements
- NEVER log only a milestone without the actual exploit command via ` + "`rt exec`" + `
- NEVER skip screenshots for web vulnerabilities
- NEVER leave findings without PoC steps in notes
- NEVER skip ` + "`rt serve`" + ` — the dashboard MUST be running before you present results to the user
- NEVER skip the wrap-up phase (report + lock)
- NEVER present final results without first running ` + "`rt serve --listen localhost:7777`" + ` and telling the user to open the dashboard

## Completion Checklist (verify ALL before finishing)

Before you say "done" or present a summary, confirm you did ALL of these:
- [ ] ` + "`rt serve --listen localhost:7777`" + ` is running (dashboard accessible)
- [ ] **Every finding has PoC notes** — run ` + "`rt query \"SELECT id, title, notes FROM findings\"`" + ` — if ANY row shows NULL notes, write them with ` + "`rt finding-note <id> \"...\"`" + ` NOW
- [ ] All findings verified WITH screenshots (` + "`rt verify-finding <id> confirmed --screenshot <file>`" + `)
- [ ] All findings have recommendations (` + "`rt recommend`" + `)
- [ ] Checklist phases checked off (` + "`rt check <id>`" + ` for each completed phase)
- [ ] ` + "`rt report --html -o report.html`" + ` was generated
- [ ] ` + "`rt verify-chain`" + ` passed
- [ ] Told the user to open http://localhost:7777 to see the dashboard`

const skillPentest = `# Pentest Target

You are a penetration tester. Use RT to capture ALL evidence.

## Target
$ARGUMENTS

## Step 1: Environment (DO THIS FIRST — every shell must have this)

` + "```bash" + `
# Windows PowerShell:
$env:RT_HOME = "$PWD\.rt"
# Linux/Mac:
export RT_HOME="$(pwd)/.rt"
` + "```" + `

**VERIFY**: Run ` + "`echo $env:RT_HOME`" + ` (PS) or ` + "`echo $RT_HOME`" + ` (bash) — it must point to THIS project directory, not ~/.rt.

## Step 2: Create Engagement + Start Dashboard

` + "```bash" + `
rt new "<name-from-target>" --client "Pentest"
rt unlock
rt scope <target-host>
rt checklist-load ptes
rt start
rt serve --listen localhost:7777 &
` + "```" + `

**Tell the user**: "Dashboard is live at http://localhost:7777 — open it for real-time tracking."

## Step 3: Test (ALL commands through rt exec)

- ` + "`rt exec -- <cmd>`" + ` — every target command (use ` + "`--`" + ` before flags like -s)
- ` + "`rt finding \"Title\" --priority <sev> --mitre <id>`" + ` — every vuln found
- ` + "`rt cred <user> <secret> --host <ip>`" + ` — every credential found
- ` + "`rt milestone \"Description\"`" + ` — every major achievement
- ` + "`rt scope-tested <host>`" + ` — after testing each host
- ` + "`rt screenshot <file>`" + ` — for visual evidence (MANDATORY for web vulns)
- ` + "`rt check <id>`" + ` — check off completed PTES phases as you go

### Evidence Quality Rules (CRITICAL)

1. **Every exploit step MUST be an ` + "`rt exec`" + ` command** — milestones alone are NOT evidence.
   If you discover a vuln, the actual curl/request that proves it MUST be captured via ` + "`rt exec`" + `.
   Bad: only ` + "`rt milestone \"Found SSTI\"`" + `. Good: ` + "`rt exec -- curl.exe -s ... (payload)`" + ` THEN milestone.

2. **If a command fails (exit code != 0), diagnose and re-run correctly.**
   On Windows PowerShell: use ` + "`curl.exe`" + ` not ` + "`curl`" + `. Quote POST data properly.
   Example: ` + "`rt exec -- curl.exe -s -X POST -d \"username=user&password=pass\" http://target/login`" + `

3. **Store ALL secrets as credentials:**
   - Passwords: ` + "`rt cred <user> <password> --host <ip>`" + `
   - JWT secrets: ` + "`rt cred JWT_SECRET <secret> --host <ip>`" + `
   - Session cookies/tokens: ` + "`rt cred admin_jwt <token-value> --host <ip>`" + `
   - API keys: ` + "`rt cred API_KEY <key> --host <ip>`" + `
   Any value that grants access MUST be stored for reuse.

4. **Write PoC steps in finding notes:**
   After creating a finding, add detailed PoC steps:
   ` + "`rt finding-note <id> \"Step 1: curl ... Step 2: ... Result: ...\"`" + `
   Or use ` + "`--note`" + ` flag: ` + "`rt finding \"Title\" --priority critical --note \"PoC: curl -s ...\"`" + `

5. **Screenshots are MANDATORY for web vulnerabilities.**
   Save browser evidence as screenshots and attach them:
   ` + "`rt screenshot <file> [evidence-id]`" + `

You decide the attack strategy. RT is your evidence pipeline, not your playbook.
Think like a pentester: enumerate, analyze, exploit, escalate, document.
**Mark checklist items as you complete each phase** (e.g. after recon: ` + "`rt check 5`" + ` through ` + "`rt check 9`" + `).

## Step 4: Wrap Up (MANDATORY — do ALL of these)

` + "```bash" + `
# 4a. PoC notes for EVERY finding (MANDATORY — run this for each finding)
rt findings                          # list all — note the IDs
rt finding-note 1 "Step 1: <exact command used>
Step 2: <what happened>
Result: <what was achieved/extracted>"
rt finding-note 2 "..."              # repeat for each finding
# PoC must include the actual commands, not just a summary!

# 4b. Verify + recommend all findings
rt verify-finding <id> confirmed --screenshot <file>  # verify with screenshot
rt recommend <id> "Fix description"  # recommend each

# 4c. Checklist
rt checklist                         # review checklist
rt check <id>                        # check off completed phases

# 4d. Generate deliverables
rt standup                           # progress summary
rt verify-chain                      # integrity check
rt report --html -o report.html      # HTML report

# 4e. Clean up
rt stop                              # stop session
rt lock                              # re-encrypt database
` + "```" + `

**STOP GATE — verify ALL before presenting results:**
1. ` + "`rt serve --listen localhost:7777`" + ` is running
2. You told the user to open http://localhost:7777
3. **Every finding has PoC notes** (` + "`rt finding-note <id> \"...\"`" + `) — run ` + "`rt query \"SELECT id, title, notes FROM findings\"`" + ` and if ANY notes is NULL, write them NOW
4. All findings verified with screenshots (` + "`rt verify-finding <id> confirmed --screenshot <file>`" + `)
5. All findings have recommendations (` + "`rt recommend <id> \"...\"`" + `)
6. Checklist items checked off (` + "`rt check <id>`" + ` for completed phases)
7. ` + "`rt report --html -o report.html`" + ` generated
8. ` + "`rt verify-chain`" + ` passed

**If ANY finding has NULL notes, you MUST write PoC notes before continuing.**
The report PoC section is generated from notes — empty notes = empty report.
`

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
