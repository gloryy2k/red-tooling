# RT — Red Team Evidence Logger

## What this is
RT is a Go CLI tool that auto-captures evidence during red team engagements and generates reports. Single binary, SQLite database with AES-256-GCM encryption, evidence hash chain integrity.

## Quick reference
```bash
rt new "ENGAGEMENT-NAME" --client "Client Corp"   # create engagement
rt unlock                                          # unlock encrypted db
rt start                                           # start interactive capture session
rt stop                                            # stop session + lock db
rt exec <command>                                  # capture single command
rt finding "Title" --priority high --mitre T1190   # create finding
rt verify-finding <id> confirmed                   # verify a finding
rt recommend <id> "Fix description"                # add recommendation
rt cred <user> <secret> --host <ip>                # store credential
rt report --html -o report.html                    # generate report
rt export ghostwriter -o findings.json             # export for Ghostwriter
rt serve --listen localhost:8080                    # start web dashboard
rt scope 10.0.0.1,10.0.0.2                         # add hosts to scope
rt scope-list                                       # list scope hosts
rt scope-tested 10.0.0.1                            # mark host tested
rt scope-untested                                   # list untested hosts
rt checklist-load ptes                              # load PTES checklist
rt checklist                                        # view checklist
rt check <id>                                       # mark checklist item done
rt search "keyword" --tag recon                     # search evidence
rt query "SELECT * FROM evidence LIMIT 10"          # read-only SQL query
rt import scan.xml                                  # import nmap/nuclei/CSV
rt standup                                          # daily progress summary
rt time                                             # session time overview
rt wipe "ENG-NAME" --confirm                        # permanently delete engagement
rt lock                                             # re-encrypt database

# Enterprise / Central Server
rt serve --listen 0.0.0.0:8443                      # start central server (TLS auto)
rt operator add alice operator                       # create operator + API key
rt remote-session start --server https://srv:8443 --key <KEY>   # start remote session
rt remote-exec --server https://srv:8443 --key <KEY> "whoami"   # exec + submit evidence
rt remote-session stop --session <ID> --server ...   # stop remote session
```

## Architecture
- **Language:** Go, single binary
- **Storage:** SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Encryption:** AES-256-GCM file-level, Argon2id KDF
- **CLI:** spf13/cobra
- **Web:** net/http + vanilla JS (no npm, no build step)

## Project structure
```
cmd/rt/           — CLI commands (one file per command group)
internal/
  agent/          — playbook runner + cost tracking
  attachments/    — file BLOB storage
  audit/          — immutable audit log
  capture/        — command execution + evidence recording
  config/         — RT home directory management
  credentials/    — credential locker + auto-parser
  crypto/         — Argon2id + AES-256-GCM + hash chain
  db/             — SQLite operations + encryption lifecycle
  engagement/     — engagement CRUD
  evidence/       — evidence bus + hash chain + auto-flag
  export/         — ghostwriter, csv, json, mitre exports
  findings/       — finding management + verify + merge
  operator/       — RBAC + API key management
  report/         — HTML + Markdown report generation
  rules/          — auto-flag rules engine (YAML regex matching)
  scope/          — scope host tracking
  remote/         — HTTP client for central server API
  search/         — evidence text search + read-only SQL
  server/         — web dashboard + central server API + WebSocket
  session/        — session management
  checklist/      — PTES/custom checklist management
  importer/       — nmap XML, nuclei JSON, CSV, text import
playbooks/        — YAML playbook templates
```

## Build
```bash
go build -o rt.exe ./cmd/rt/
```

## Key design decisions
- File-level encryption (.db.enc ↔ .db) instead of SQLCipher to avoid CGO/MinGW on Windows
- Cross-process unlock state: plain .db file exists = unlocked
- Passphrase cached in XOR-obfuscated .key file for process isolation
- Evidence hash chain: SHA-256(timestamp|action|input|output|exit_code|tags|operator)
- Credentials double-encrypted: AES-256-GCM inside already-encrypted database
- All mutations audit-logged with operator, action, target, detail
- Auto-flag rules engine matches command output → auto-tag, auto-priority, auto-MITRE
- Auto-cred parser extracts credentials from secretsdump/mimikatz/kerberoast output

## When working on this codebase
- Always run `go build -o rt.exe ./cmd/rt/` to verify compilation
- Test with `bash tests/test_m<N>.sh` scripts in tests/ folder
- Set `RT_HOME` env var to avoid polluting real ~/.rt during testing
- The database must be unlocked before most commands work
- Evidence chain integrity is critical — never modify evidence rows directly
- `rt new` auto-generates agent context (CLAUDE.md + skills) in ~/.rt/engagements/<name>/
- Report templates are customizable via ~/.rt/templates/<name>.yml (company_name, accent_color, etc.)
- Auto-flag rules support real regex, match conditions (all/any), not_patterns exclusions, confidence scores
- Dashboard at `rt serve` supports full CRUD: create/verify findings, manage scope, toggle checklist, all from browser
- Central server mode: `rt serve --listen 0.0.0.0:8443` exposes REST API for remote agents
- Remote agents use `rt remote-exec` / `rt remote-session` to POST evidence to central server
- Server API: POST /api/evidence, POST /api/sessions, POST /api/creds — all with RBAC + WebSocket broadcast
- Environment vars RT_SERVER and RT_API_KEY configure remote agent connections
