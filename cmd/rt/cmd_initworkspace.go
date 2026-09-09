package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initWorkspaceCmd = &cobra.Command{
	Use:   "init-workspace",
	Short: "Generate AI agent context (CLAUDE.md + skills) in current directory",
	Long: `Creates CLAUDE.md and .claude/commands/ skills so any AI coding agent
(Claude Code, Cursor, etc.) understands how to use RT for pentesting.

Run this in your pentest working directory before starting a session.
The agent will auto-discover RT capabilities and follow the correct workflow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		force, _ := cmd.Flags().GetBool("force")

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if _, err := os.Stat(claudeMD); err == nil && !force {
			return fmt.Errorf("CLAUDE.md already exists (use --force to overwrite)")
		}

		if err := os.WriteFile(claudeMD, []byte(agentClaudeMD), 0644); err != nil {
			return fmt.Errorf("write CLAUDE.md: %w", err)
		}
		fmt.Printf("  Created: %s\n", claudeMD)

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
			if _, err := os.Stat(p); err == nil && !force {
				fmt.Printf("  Skipped: %s (exists)\n", p)
				continue
			}
			if err := os.WriteFile(p, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
			fmt.Printf("  Created: %s\n", p)
		}

		fmt.Println()
		fmt.Println("  Workspace ready. An AI agent in this directory will:")
		fmt.Println("    1. Read CLAUDE.md → understand RT capabilities")
		fmt.Println("    2. Run ./rt --help → discover all commands")
		fmt.Println("    3. Use /pentest <target> → full attack workflow")
		fmt.Println("    4. Use /report → generate final report")
		fmt.Println()
		fmt.Println("  The agent only needs a target. Everything else is automatic.")

		return nil
	},
}

func init() {
	initWorkspaceCmd.Flags().Bool("force", false, "Overwrite existing files")
}

const agentClaudeMD = `# Pentest Workstation — RT Evidence Logger

This folder contains ` + "`rt`" + ` (or ` + "`rt.exe`" + ` on Windows) — a red team evidence capture tool.
You are the pentester. The tool is your evidence pipeline.

## MANDATORY: Learn the tool first

Before ANY pentest activity, run:
` + "```bash" + `
./rt --help
` + "```" + `

Read every subcommand. Then check key ones you'll use most:
` + "```bash" + `
./rt exec --help
./rt finding --help
./rt cred --help
./rt scope --help
./rt serve --help
./rt report --help
` + "```" + `

## The One Rule

**Every command you execute against a target MUST go through ` + "`rt exec`" + `.**

` + "```bash" + `
# WRONG — evidence lost
curl http://target/api/users

# RIGHT — captured, auto-flagged, hash-chained
./rt exec curl http://target/api/users
` + "```" + `

RT auto-detects credentials, admin access, RCE indicators, SQL injection, ADCS,
kerberoast output and more. It builds a tamper-proof SHA-256 hash chain of everything.

## Passphrase (non-interactive)

` + "```bash" + `
echo -e "rtpass\nrtpass" | ./rt new "NAME" --client "Client"  # create (2x)
echo "rtpass" | ./rt unlock                                    # unlock (1x)
# Everything else works without passphrase once unlocked
` + "```" + `

## Record findings immediately when discovered

| Discovery | Command |
|-----------|---------|
| Vulnerability | ` + "`./rt finding \"Title\" --priority critical --mitre T1190`" + ` |
| Credentials | ` + "`./rt cred <user> <secret> --host <ip>`" + ` |
| Achievement | ` + "`./rt milestone \"Got admin shell\"`" + ` |
| Observation | ` + "`./rt note \"Interesting detail\"`" + ` |
| Host tested | ` + "`./rt scope-tested <host>`" + ` |
| Checklist done | ` + "`./rt check <item-id>`" + ` |

## Dashboard

` + "`./rt serve --listen localhost:7777`" + ` starts a live web dashboard with
evidence feed, findings, credentials, scope progress, and checklist.

## Workflow

1. ` + "`/pentest <target>`" + ` — full automated pentest workflow
2. ` + "`/report`" + ` — verify findings + generate report

## What NOT to do

- NEVER run commands without ` + "`rt exec`" + ` — evidence is lost
- NEVER forget ` + "`rt cred`" + ` when you find credentials
- NEVER forget ` + "`rt finding`" + ` when you discover a vulnerability
- NEVER forget ` + "`rt milestone`" + ` for major achievements (shell, admin, flag)
- NEVER leave findings unverified — always ` + "`rt verify-finding`" + `
- NEVER generate report without recommendations — always ` + "`rt recommend`" + `
`

const skillPentest = `# Pentest Target

You are a penetration tester. The user has given you a target.
Set up RT, attack the target, capture ALL evidence through RT, and produce a report.

## Target
$ARGUMENTS

## Phase 0: Learn the tool (MANDATORY — do not skip)

` + "```bash" + `
./rt --help
./rt exec --help
./rt finding --help
./rt cred --help
` + "```" + `

Read and understand every subcommand before proceeding.

## Phase 1: Setup

` + "```bash" + `
export RT_HOME=./.rt
echo -e "rtpass\nrtpass" | ./rt new "<derive-name-from-target>" --client "Pentest"
echo "rtpass" | ./rt unlock
./rt scope <target-host-or-ip>
./rt checklist-load ptes
./rt serve --listen localhost:7777 &
` + "```" + `

Tell user: dashboard at http://localhost:7777

## Phase 2: Recon (ALL through rt exec)

` + "```bash" + `
./rt exec curl -sI http://<target>
./rt exec curl -s http://<target>/robots.txt
./rt exec curl -s http://<target>/.git/HEAD
./rt exec curl -s http://<target>/.env
./rt exec curl -s http://<target>/
` + "```" + `

Analyze responses. Follow interesting paths. For each discovery:
- Vulnerability → ` + "`./rt finding \"Title\" --priority <level> --mitre <id>`" + `
- Credentials → ` + "`./rt cred <user> <secret> --host <target>`" + `
- Key discovery → ` + "`./rt milestone \"Description\"`" + `

After recon: ` + "`./rt scope-tested <target>`" + ` + check checklist items.

## Phase 3: Exploit (ALL through rt exec)

Based on recon, exploit vulnerabilities. Common patterns:
- Auth bypass (default creds, JWT manipulation, session tokens)
- Injection (SQLi, SSTI, command injection, XSS)
- Information disclosure (.git, .env, config, source code)
- Privilege escalation (token forge, IDOR, role manipulation)
- File inclusion (LFI/RFI)

For EACH successful exploit:
1. ` + "`./rt finding \"Title\" --priority critical --mitre <id>`" + `
2. ` + "`./rt cred`" + ` if credentials found
3. ` + "`./rt milestone`" + ` for major achievements

## Phase 4: Wrap up

` + "```bash" + `
./rt standup
./rt verify-chain
` + "```" + `

Present summary to user. Suggest running /report for final output.
`

const skillReport = `# Generate Pentest Report

Verify all findings, add recommendations, and generate the final report.

## Instructions

1. List all findings:
` + "```bash" + `
./rt findings
` + "```" + `

2. For each unverified finding, verify it:
` + "```bash" + `
./rt verify-finding <id> confirmed --note "How it was verified"
` + "```" + `

3. For each finding without a recommendation, add one:
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

6. Present summary to user: findings count by severity, scope coverage, time spent.
`
