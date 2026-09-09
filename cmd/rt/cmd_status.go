package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/db"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current engagement status",
	RunE: func(cmd *cobra.Command, args []string) error {
		engName := config.GetActiveEngagement()
		if engName == "" {
			fmt.Println("  No active engagement.")
			fmt.Println("  Run 'rt new \"NAME\"' to create one.")
			return nil
		}

		fmt.Printf("\n  Engagement: %s\n", engName)

		if !db.IsUnlocked(engName) {
			fmt.Println("  Status: locked")
			fmt.Println("  Run 'rt unlock' to access")
			fmt.Println()
			return nil
		}

		fmt.Println("  Status: unlocked")

		database, err := db.OpenPlain(engName)
		if err != nil {
			return err
		}

		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		// Active session
		sess, _ := session.GetActive(database, engID)
		if sess != nil {
			count := evidence.CountBySession(database, sess.ID)
			fmt.Printf("  Active session: %s (%d entries)\n", sess.Name, count)
		} else {
			fmt.Println("  Active session: none")
		}

		// Total evidence
		var total int
		database.QueryRow(
			`SELECT COUNT(*) FROM evidence e JOIN sessions s ON e.session_id = s.id
			 WHERE s.engagement_id = ? AND e.is_deleted = 0`, engID,
		).Scan(&total)
		fmt.Printf("  Total evidence: %d entries\n", total)

		// Sessions
		sessions, _ := session.List(database, engID)
		fmt.Printf("  Sessions: %d\n", len(sessions))

		fmt.Println()
		return nil
	},
}
