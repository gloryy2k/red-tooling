package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/checklist"
)

var checklistCmd = &cobra.Command{
	Use:   "checklist",
	Short: "Show engagement checklist",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		items, err := checklist.List(database, engID)
		if err != nil {
			return err
		}

		if len(items) == 0 {
			fmt.Println("  No checklist items. Use 'rt checklist-load ptes' to load a template.")
			return nil
		}

		done, total := checklist.Stats(database, engID)
		fmt.Printf("  Checklist: %d/%d complete\n\n", done, total)

		currentCat := ""
		for _, it := range items {
			if it.Category != currentCat {
				currentCat = it.Category
				fmt.Printf("\n  [%s]\n", currentCat)
			}
			status := "[ ]"
			if it.Done {
				status = "[x]"
			}
			fmt.Printf("  %s #%d %s", status, it.ID, it.Item)
			if it.EvidenceID > 0 {
				fmt.Printf("  (evidence #%d)", it.EvidenceID)
			}
			fmt.Println()
		}
		return nil
	},
}

var checklistLoadCmd = &cobra.Command{
	Use:   "checklist-load <template>",
	Short: "Load a checklist template (ptes)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		switch args[0] {
		case "ptes":
			count, err := checklist.LoadPTES(database, engID)
			if err != nil {
				return err
			}
			fmt.Printf("  Loaded PTES checklist: %d items\n", count)
		default:
			return fmt.Errorf("unknown template: %s (available: ptes)", args[0])
		}
		return nil
	},
}

var checklistAddCmd = &cobra.Command{
	Use:   "checklist-add <category> <item>",
	Short: "Add a custom checklist item",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		if err := checklist.Add(database, engID, args[0], args[1]); err != nil {
			return err
		}

		fmt.Printf("  Added: [%s] %s\n", args[0], args[1])
		return nil
	},
}

var checkCmd = &cobra.Command{
	Use:   "check <checklist-id>",
	Short: "Mark a checklist item as done",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid checklist ID: %s", args[0])
		}

		operator := getOperator()
		evidenceID, _ := cmd.Flags().GetInt64("evidence")

		if err := checklist.Check(database, id, operator, evidenceID); err != nil {
			return err
		}

		fmt.Printf("  Checked off item #%d\n", id)
		return nil
	},
}

var uncheckCmd = &cobra.Command{
	Use:   "uncheck <checklist-id>",
	Short: "Uncheck a checklist item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid checklist ID: %s", args[0])
		}

		if err := checklist.Uncheck(database, id); err != nil {
			return err
		}

		fmt.Printf("  Unchecked item #%d\n", id)
		return nil
	},
}

func init() {
	checkCmd.Flags().Int64("evidence", 0, "Link to evidence ID")
}
