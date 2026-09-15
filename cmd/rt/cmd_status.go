package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/db"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/remote"
	"github.com/user/rt/internal/session"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current engagement and connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()

		if state, err := remote.LoadState(); err == nil {
			fmt.Printf("  Remote: joined %s\n", state.ServerURL)
			fmt.Printf("  Session: %s\n", state.SessionID)
			fmt.Printf("  Operator: %s\n", state.Operator)
			if state.Insecure {
				fmt.Printf("  TLS: insecure (skip verify)\n")
			}
			qLen := remote.QueueLen()
			if qLen > 0 {
				fmt.Printf("  Queue: %d item(s) pending sync\n", qLen)
			}
		} else {
			fmt.Println("  Remote: not joined")
		}

		engName := config.GetActiveEngagement()
		if engName == "" {
			fmt.Println("  Local: no active engagement")
			fmt.Println()
			return nil
		}

		fmt.Printf("  Local: %s\n", engName)

		if !db.IsUnlocked(engName) {
			fmt.Println("  DB: locked")
			fmt.Println()
			return nil
		}

		fmt.Println("  DB: unlocked")

		database, err := db.OpenPlain(engName)
		if err != nil {
			return err
		}

		engID := resolveEngID(database, engName)

		sess, _ := session.GetActive(database, engID)
		if sess != nil {
			count := evidence.CountBySession(database, sess.ID)
			fmt.Printf("  Active session: %s (%d entries)\n", sess.Name, count)
		} else {
			fmt.Println("  Active session: none")
		}

		var total int
		database.QueryRow(
			`SELECT COUNT(*) FROM evidence e JOIN sessions s ON e.session_id = s.id
			 WHERE s.engagement_id = ? AND e.is_deleted = 0`, engID,
		).Scan(&total)
		fmt.Printf("  Total evidence: %d entries\n", total)

		sessions, _ := session.List(database, engID)
		fmt.Printf("  Sessions: %d\n", len(sessions))

		fmt.Println()
		return nil
	},
}
