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

## Rules

1. **Every command against a target MUST go through ` + "`rt exec`" + `.**
   ` + "```" + `
   # WRONG: evidence lost
   curl http://target/api/users

   # RIGHT: captured, auto-flagged, hash-chained
   ./rt exec curl http://target/api/users
   ` + "```" + `

2. **Record findings immediately.** When you discover a vulnerability:
   ` + "`./rt finding \"Title\" --priority <level> --mitre <id>`" + `

3. **Record credentials immediately.** When you find creds:
   ` + "`./rt cred <user> <secret> --host <ip>`" + `

4. **Mark milestones.** For major achievements (shell, admin, flag):
   ` + "`./rt milestone \"Description\"`" + `

5. **Track scope.** Mark hosts tested: ` + "`./rt scope-tested <host>`" + `

6. **Verify findings.** Before reporting, verify each finding:
   ` + "`./rt verify-finding <id> confirmed --screenshot <file>`" + ` (visual evidence)
   ` + "`./rt verify-finding <id> confirmed --no-screenshot`" + ` (non-visual)

7. **Add recommendations.** Every finding needs remediation guidance:
   ` + "`./rt recommend <id> \"Specific fix\"`" + `

## Quick Reference

Run ` + "`./rt --help`" + ` to see all available commands.

| Action | Command |
|--------|---------|
| Execute + capture | ` + "`./rt exec <cmd>`" + ` |
| Create finding | ` + "`./rt finding \"Title\" --priority high --mitre T1190`" + ` |
| Store credential | ` + "`./rt cred <user> <secret> --host <ip>`" + ` |
| Mark milestone | ` + "`./rt milestone \"Got admin\"`" + ` |
| Attach screenshot | ` + "`./rt screenshot <file> [evidence-id]`" + ` |
| Verify finding | ` + "`./rt verify-finding <id> confirmed`" + ` |
| Add recommendation | ` + "`./rt recommend <id> \"Fix\"`" + ` |
| Dashboard | ` + "`./rt serve --listen localhost:7777`" + ` |
| Generate report | ` + "`./rt report --html -o report.html`" + ` |

## What NOT to do

- NEVER run target commands outside ` + "`rt exec`" + `
- NEVER leave findings unverified or without recommendations
- NEVER skip ` + "`rt cred`" + ` when credentials are found
- NEVER skip ` + "`rt milestone`" + ` for major achievements`

const skillPentest = `# Pentest Target

You are a penetration tester. Use RT to capture ALL evidence.

## Target
$ARGUMENTS

## Mandatory Setup

Before ANY pentesting activity, learn the tool:
` + "```bash" + `
./rt --help
./rt exec --help
./rt finding --help
./rt cred --help
` + "```" + `

Then set up the engagement:
` + "```bash" + `
export RT_HOME=./.rt
echo -e "rtpass\nrtpass" | ./rt new "<name>" --client "Pentest"
echo "rtpass" | ./rt unlock
./rt scope <target>
./rt checklist-load ptes
./rt start
` + "```" + `

## Rules During Testing

- **ALL** commands against the target go through ` + "`./rt exec`" + `
- **Immediately** record findings: ` + "`./rt finding \"Title\" --priority <sev> --mitre <id>`" + `
- **Immediately** record credentials: ` + "`./rt cred <user> <secret> --host <ip>`" + `
- **Immediately** mark milestones: ` + "`./rt milestone \"What happened\"`" + `
- **Attach screenshots** for visual evidence: ` + "`./rt screenshot <file>`" + `
- **Track scope**: ` + "`./rt scope-tested <host>`" + ` after testing each host
- **Check items**: ` + "`./rt check <id>`" + ` as you complete checklist phases

## Strategy

You decide the attack strategy. RT is your evidence pipeline, not your playbook.
Think like a pentester: enumerate, analyze, exploit, escalate, document.

## Wrap Up

When done testing:
` + "```bash" + `
./rt standup
./rt verify-chain
./rt stop
` + "```" + `

Present summary to user. Suggest ` + "`/report`" + ` for final output.
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
