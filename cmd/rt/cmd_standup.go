package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/checklist"
	"github.com/user/rt/internal/scope"
)

var standupCmd = &cobra.Command{
	Use:   "standup",
	Short: "Show daily standup summary (recent activity, scope, checklist)",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		fmt.Println("  === Daily Standup ===")
		fmt.Println()

		// Scope progress
		total, tested := scope.Stats(database, engID)
		if total > 0 {
			pct := float64(tested) / float64(total) * 100
			fmt.Printf("  Scope:     %d/%d hosts tested (%.0f%%)\n", tested, total, pct)
		} else {
			fmt.Println("  Scope:     no hosts defined")
		}

		// Checklist progress
		done, clTotal := checklist.Stats(database, engID)
		if clTotal > 0 {
			pct := float64(done) / float64(clTotal) * 100
			fmt.Printf("  Checklist: %d/%d items done (%.0f%%)\n", done, clTotal, pct)
		} else {
			fmt.Println("  Checklist: no items defined")
		}

		// Evidence count last 24h
		var evCount int
		database.QueryRow(
			`SELECT COUNT(*) FROM evidence e
			 JOIN sessions s ON e.session_id = s.id
			 WHERE s.engagement_id = ? AND e.is_deleted = 0
			 AND e.timestamp >= datetime('now', '-1 day')`, engID).Scan(&evCount)
		fmt.Printf("  Evidence:  %d entries in last 24h\n", evCount)

		// Findings summary
		var fTotal, fCritical, fHigh int
		database.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ?`, engID).Scan(&fTotal)
		database.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ? AND priority = 'critical'`, engID).Scan(&fCritical)
		database.QueryRow(`SELECT COUNT(*) FROM findings WHERE engagement_id = ? AND priority = 'high'`, engID).Scan(&fHigh)
		fmt.Printf("  Findings:  %d total (%d critical, %d high)\n", fTotal, fCritical, fHigh)

		// Credentials found
		var credCount int
		database.QueryRow(`SELECT COUNT(*) FROM credentials WHERE engagement_id = ?`, engID).Scan(&credCount)
		fmt.Printf("  Creds:     %d captured\n", credCount)

		// Recent audit activity
		fmt.Println()
		fmt.Println("  --- Recent Activity ---")
		rows, err := database.Query(
			`SELECT timestamp, operator, action, target_type, COALESCE(target_id, '')
			 FROM audit_log ORDER BY timestamp DESC LIMIT 10`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ts, op, action, ttype, tid string
				rows.Scan(&ts, &op, &action, &ttype, &tid)
				if len(ts) > 19 {
					ts = ts[:19]
				}
				fmt.Printf("  %s  %-12s %-24s %s/%s\n", ts, op, action, ttype, tid)
			}
		}

		return nil
	},
}

var timeCmd = &cobra.Command{
	Use:   "time",
	Short: "Show session durations and total engagement time",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		rows, err := database.Query(
			`SELECT name, source, started_at, COALESCE(ended_at, ''), status
			 FROM sessions WHERE engagement_id = ?
			 ORDER BY started_at`, engID)
		if err != nil {
			return err
		}
		defer rows.Close()

		fmt.Println("  Sessions:")
		fmt.Printf("  %-30s %-14s %-20s %-8s\n", "Name", "Source", "Started", "Status")
		fmt.Printf("  %s\n", strings.Repeat("-", 76))

		for rows.Next() {
			var name, source, started, ended, status string
			rows.Scan(&name, &source, &started, &ended, &status)
			if len(started) > 19 {
				started = started[:19]
			}
			fmt.Printf("  %-30s %-14s %-20s %-8s\n", name, source, started, status)
		}

		// Total evidence count
		var totalEvidence int
		database.QueryRow(
			`SELECT COUNT(*) FROM evidence e JOIN sessions s ON e.session_id = s.id
			 WHERE s.engagement_id = ? AND e.is_deleted = 0`, engID).Scan(&totalEvidence)
		fmt.Printf("\n  Total evidence entries: %d\n", totalEvidence)

		return nil
	},
}

var wipeCmd = &cobra.Command{
	Use:   "wipe <engagement-name>",
	Short: "Permanently delete an engagement and its database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engName := args[0]

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			return fmt.Errorf("this permanently deletes ALL data for %q. Add --confirm to proceed", engName)
		}

		rtHome := os.Getenv("RT_HOME")
		if rtHome == "" {
			home, _ := os.UserHomeDir()
			rtHome = filepath.Join(home, ".rt")
		}

		engDir := filepath.Join(rtHome, "engagements", engName)
		if _, err := os.Stat(engDir); os.IsNotExist(err) {
			return fmt.Errorf("engagement %q not found", engName)
		}

		// Try to audit-log the wipe if possible
		dbPath := filepath.Join(engDir, engName+".db")
		if _, err := os.Stat(dbPath); err == nil {
			if d, err := sql.Open("sqlite", dbPath); err == nil {
				audit.Log(d, getOperator(), "engagement.wipe", "engagement", engName, nil)
				d.Close()
			}
		}

		if err := os.RemoveAll(engDir); err != nil {
			return fmt.Errorf("wipe failed: %w", err)
		}

		fmt.Printf("  Engagement %q permanently deleted\n", engName)
		return nil
	},
}

func init() {
	wipeCmd.Flags().Bool("confirm", false, "Confirm permanent deletion")
}
