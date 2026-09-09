package agent

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/capture"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

type RunResult struct {
	Playbook    string
	StepsRun    int
	StepsTotal  int
	Skipped     int
	Failed      int
	Duration    time.Duration
	EvidenceIDs []int64
}

// RunPlaybook executes a playbook's steps sequentially with dependency resolution.
func RunPlaybook(db *sql.DB, engID string, pb *Playbook, vars map[string]string) (*RunResult, error) {
	if err := ResolveInputs(pb, vars); err != nil {
		return nil, err
	}

	// Create agent session
	sessID, _ := crypto.RandomHex(8)
	sessName := fmt.Sprintf("agent-%s-%s", pb.Name, time.Now().Format("20060102-150405"))
	if err := session.Create(db, sessID, engID, sessName, "agent:playbook", ""); err != nil {
		return nil, fmt.Errorf("create agent session: %w", err)
	}

	// Create work directory
	workDir := filepath.Join(os.TempDir(), "rt-agent-"+sessID)
	os.MkdirAll(workDir, 0700)
	vars["work_dir"] = workDir

	audit.Log(db, "agent", "agent.playbook.start", "session", sessID,
		map[string]string{"playbook": pb.Name, "steps": fmt.Sprintf("%d", len(pb.Steps))})

	fmt.Printf("\n  [agent] Running playbook: %s (%d steps)\n", pb.Name, len(pb.Steps))
	fmt.Printf("  [agent] Session: %s\n", sessName)
	fmt.Printf("  [agent] Work dir: %s\n\n", workDir)

	start := time.Now()
	result := &RunResult{
		Playbook:   pb.Name,
		StepsTotal: len(pb.Steps),
	}

	completed := make(map[string]bool)

	for _, step := range pb.Steps {
		// Check dependency
		if step.DependsOn != "" && !completed[step.DependsOn] {
			fmt.Printf("  [skip] %s — dependency %q not completed\n", step.Name, step.DependsOn)
			result.Skipped++
			continue
		}

		// Evaluate condition
		if !EvalCondition(step.Condition) {
			fmt.Printf("  [skip] %s — condition not met\n", step.Name)
			result.Skipped++
			continue
		}

		// Re-substitute vars (work_dir may have been added)
		cmd := substituteVars(step.Cmd, vars)

		fmt.Printf("  [step] %s: %s\n", step.Name, cmd)

		ev, flagResult, err := capture.RunCommand(db, sessID, engID, "agent", cmd, workDir)
		if err != nil {
			fmt.Printf("  [fail] %s: %v\n", step.Name, err)
			result.Failed++
			if step.OnFail != "continue" {
				break
			}
			continue
		}

		// Update evidence tags from step
		if len(step.Tags) > 0 {
			existingTags := ev.Tags
			allTags := append(existingTags, step.Tags...)
			tagsJSON := fmt.Sprintf(`["%s"]`, strings.Join(allTags, `","`))
			db.Exec(`UPDATE evidence SET tags = ? WHERE id = ?`, tagsJSON, ev.ID)
		}

		result.EvidenceIDs = append(result.EvidenceIDs, ev.ID)
		result.StepsRun++
		completed[step.Name] = true

		if ev.ExitCode != 0 {
			fmt.Printf("  [warn] %s exited with code %d\n", step.Name, ev.ExitCode)
			if step.OnFail != "continue" {
				result.Failed++
				break
			}
		}

		if flagResult != nil {
			evidence.PrintAutoFlagResult(flagResult)
		}
	}

	result.Duration = time.Since(start)

	// Stop session
	session.Stop(db, sessID, "agent")

	audit.Log(db, "agent", "agent.playbook.complete", "session", sessID,
		map[string]interface{}{
			"playbook":  pb.Name,
			"steps_run": result.StepsRun,
			"skipped":   result.Skipped,
			"failed":    result.Failed,
			"duration":  result.Duration.String(),
		})

	fmt.Printf("\n  [agent] Playbook complete: %d/%d steps, %d skipped, %d failed (%s)\n\n",
		result.StepsRun, result.StepsTotal, result.Skipped, result.Failed, result.Duration.Round(time.Millisecond))

	return result, nil
}

// RunScript executes a script file and captures all output as evidence.
func RunScript(db *sql.DB, engID, scriptPath string) (*RunResult, error) {
	sessID, _ := crypto.RandomHex(8)
	sessName := fmt.Sprintf("agent-script-%s", time.Now().Format("20060102-150405"))
	if err := session.Create(db, sessID, engID, sessName, "agent:script", ""); err != nil {
		return nil, fmt.Errorf("create agent session: %w", err)
	}

	cwd, _ := os.Getwd()
	start := time.Now()

	fmt.Printf("\n  [agent] Running script: %s\n", scriptPath)
	fmt.Printf("  [agent] Session: %s\n\n", sessName)

	ev, flagResult, err := capture.RunCommand(db, sessID, engID, "agent", scriptPath, cwd)
	if err != nil {
		return nil, err
	}

	if flagResult != nil {
		evidence.PrintAutoFlagResult(flagResult)
	}

	session.Stop(db, sessID, "agent")

	return &RunResult{
		Playbook:    filepath.Base(scriptPath),
		StepsRun:    1,
		StepsTotal:  1,
		Duration:    time.Since(start),
		EvidenceIDs: []int64{ev.ID},
	}, nil
}

// RunSingleCommand executes a single command as an agent.
func RunSingleCommand(db *sql.DB, engID, command string) (*RunResult, error) {
	sessID, _ := crypto.RandomHex(8)
	sessName := fmt.Sprintf("agent-exec-%s", time.Now().Format("20060102-150405"))
	if err := session.Create(db, sessID, engID, sessName, "agent:exec", ""); err != nil {
		return nil, fmt.Errorf("create agent session: %w", err)
	}

	cwd, _ := os.Getwd()
	start := time.Now()

	fmt.Printf("\n  [agent] Executing: %s\n\n", command)

	ev, flagResult, err := capture.RunCommand(db, sessID, engID, "agent", command, cwd)
	if err != nil {
		return nil, err
	}

	if flagResult != nil {
		evidence.PrintAutoFlagResult(flagResult)
	}

	session.Stop(db, sessID, "agent")

	return &RunResult{
		Playbook:    "exec",
		StepsRun:    1,
		StepsTotal:  1,
		Duration:    time.Since(start),
		EvidenceIDs: []int64{ev.ID},
	}, nil
}
