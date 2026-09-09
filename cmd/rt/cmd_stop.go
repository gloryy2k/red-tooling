package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/db"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the active capture session",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		sess, err := session.GetActive(database, engID)
		if err != nil {
			return err
		}
		if sess == nil {
			return fmt.Errorf("no active session")
		}

		count := evidence.CountBySession(database, sess.ID)
		if err := session.Stop(database, sess.ID, "system"); err != nil {
			return err
		}
		session.ClearActiveSession()

		// Lock the database
		db.Close()
		if err := db.Lock(engName); err != nil {
			fmt.Printf("  Warning: could not re-encrypt: %v\n", err)
		}

		fmt.Printf("\n  Session %q stopped.\n", sess.Name)
		fmt.Printf("  Evidence entries: %d\n", count)
		fmt.Println("  Database encrypted and saved.")
		fmt.Println()
		return nil
	},
}
