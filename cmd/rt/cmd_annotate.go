package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

var tagCmd = &cobra.Command{
	Use:   "tag <message>",
	Short: "Tag the current session with a note",
	Args:  cobra.MinimumNArgs(1),
	RunE:  annotateCmd("tag", "info"),
}

var milestoneCmd = &cobra.Command{
	Use:   "milestone <message>",
	Short: "Record a milestone achievement",
	Args:  cobra.MinimumNArgs(1),
	RunE:  annotateCmd("milestone", "high"),
}

var bookmarkCmd = &cobra.Command{
	Use:   "bookmark <message>",
	Short: "Bookmark something to revisit later",
	Args:  cobra.MinimumNArgs(1),
	RunE:  annotateCmd("bookmark", "info"),
}

var noteCmd = &cobra.Command{
	Use:   "note <message>",
	Short: "Add a free-text note",
	Args:  cobra.MinimumNArgs(1),
	RunE:  annotateCmd("note", "info"),
}

func annotateCmd(action, defaultPriority string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		database, sessID, _, err := getActiveSessionDB()
		if err != nil {
			return err
		}

		msg := strings.Join(args, " ")
		cwd, _ := os.Getwd()
		operator := getOperator()

		ev, err := evidence.Insert(database, sessID, action, msg, "", 0, 0, cwd, nil, defaultPriority, operator)
		if err != nil {
			return err
		}

		icons := map[string]string{
			"tag":       "# ",
			"milestone": "** ",
			"bookmark":  ">> ",
			"note":      ".. ",
		}
		icon := icons[action]
		fmt.Printf("  %s%s (evidence #%d)\n", icon, msg, ev.ID)
		return nil
	}
}

// getActiveSessionDB returns (database, sessionID, engagementID, error).
func getActiveSessionDB() (*sql.DB, string, string, error) {
	database, engName, err := requireDB()
	if err != nil {
		return nil, "", "", err
	}

	engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

	sessID := session.GetActiveSessionID()
	if sessID == "" {
		newID, _ := crypto.RandomHex(8)
		name := fmt.Sprintf("cli-%s", time.Now().Format("20060102-150405"))
		if err := session.Create(database, newID, engID, name, "human", ""); err != nil {
			return nil, "", "", fmt.Errorf("auto-create session: %w", err)
		}
		session.SetActiveSession(newID)
		sessID = newID
		fmt.Printf("  [rt] Auto-created session: %s\n", name)
	}

	return database, sessID, engID, nil
}

func getOperator() string {
	op := os.Getenv("USER")
	if op == "" {
		op = os.Getenv("USERNAME")
	}
	if op == "" {
		op = "operator"
	}
	return op
}
