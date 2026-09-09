package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/attachments"
)

var attachCmd = &cobra.Command{
	Use:   "attach <evidence-id> <file-path>",
	Short: "Attach a file to an evidence entry",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		evID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid evidence ID: %s", args[0])
		}

		filePath := args[1]
		caption, _ := cmd.Flags().GetString("caption")
		operator := getOperator()

		a, err := attachments.Store(database, evID, filePath, caption, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Attached: %s (%s, %d bytes) → evidence #%d\n", a.Filename, a.Filetype, a.SizeBytes, evID)
		return nil
	},
}

var attachmentsCmd = &cobra.Command{
	Use:   "attachments [evidence-id]",
	Short: "List attachments",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		var list []attachments.Attachment

		if len(args) > 0 {
			evID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid evidence ID: %s", args[0])
			}
			list, err = attachments.ListByEvidence(database, evID)
			if err != nil {
				return err
			}
		} else {
			engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
			list, err = attachments.ListByEngagement(database, engID)
			if err != nil {
				return err
			}
		}

		if len(list) == 0 {
			fmt.Println("  No attachments.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-4s %-6s %-30s %-12s %-10s %s\n", "ID", "EV#", "FILENAME", "TYPE", "SIZE", "CAPTION")
		fmt.Printf("  %-4s %-6s %-30s %-12s %-10s %s\n",
			strings.Repeat("-", 4), strings.Repeat("-", 6), strings.Repeat("-", 30),
			strings.Repeat("-", 12), strings.Repeat("-", 10), strings.Repeat("-", 20))

		for _, a := range list {
			sizeStr := formatSize(a.SizeBytes)
			caption := a.Caption
			if len(caption) > 30 {
				caption = caption[:27] + "..."
			}
			fmt.Printf("  %-4d %-6d %-30s %-12s %-10s %s\n",
				a.ID, a.EvidenceID, truncate(a.Filename, 30), a.Filetype, sizeStr, caption)
		}
		fmt.Printf("\n  %d attachment(s)\n\n", len(list))
		return nil
	},
}

var exportAttachCmd = &cobra.Command{
	Use:   "export-attach <attachment-id> [output-dir]",
	Short: "Export an attachment to disk",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid attachment ID: %s", args[0])
		}

		outDir := "."
		if len(args) > 1 {
			outDir = args[1]
		}

		operator := getOperator()
		path, err := attachments.Export(database, id, outDir, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Exported: %s\n", path)
		return nil
	},
}

func init() {
	attachCmd.Flags().String("caption", "", "Caption for the attachment")
}

