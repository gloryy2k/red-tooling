package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/attachments"
	"github.com/user/rt/internal/evidence"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot <file-path> [evidence-id]",
	Short: "Attach a screenshot to evidence (defaults to latest)",
	Long:  `Store a screenshot (PNG/JPG/GIF) as an attachment. If no evidence ID is given, attaches to the most recent evidence entry.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		filePath := args[0]
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		caption, _ := cmd.Flags().GetString("caption")
		operator := getOperator()

		var evID int64
		if len(args) > 1 {
			evID, err = strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid evidence ID: %s", args[1])
			}
		} else {
			evID, err = evidence.LatestID(database, engName)
			if err != nil {
				return fmt.Errorf("no evidence found — specify an evidence ID explicitly")
			}
		}

		a, err := attachments.Store(database, evID, filePath, caption, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Screenshot attached: %s (%d bytes) → evidence #%d\n", a.Filename, a.SizeBytes, evID)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().String("caption", "", "Caption for the screenshot")
}
