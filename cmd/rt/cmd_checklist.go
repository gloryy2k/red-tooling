package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/checklist"
	"github.com/user/rt/internal/remote"
)

var checklistCmd = &cobra.Command{
	Use:   "checklist",
	Short: "Show engagement checklist",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			result, cerr := client.GetChecklist()
			if cerr != nil {
				return fmt.Errorf("get checklist (remote): %w", cerr)
			}
			items, _ := result["items"].([]interface{})
			if len(items) == 0 {
				fmt.Println("  No checklist items. Use 'rt checklist-load ptes' to load a template.")
				return nil
			}
			total := toInt(result["total"])
			done := toInt(result["done"])
			fmt.Printf("  Checklist: %d/%d complete (remote)\n\n", done, total)
			currentCat := ""
			for _, it := range items {
				if im, ok := it.(map[string]interface{}); ok {
					cat := fmt.Sprintf("%v", im["category"])
					if cat != currentCat {
						currentCat = cat
						fmt.Printf("\n  [%s]\n", currentCat)
					}
					mark := "[ ]"
					if cb, ok := im["done"].(bool); ok && cb {
						mark = "[x]"
					}
					fmt.Printf("  %s #%v %v\n", mark, im["id"], im["item"])
				}
			}
			return nil
		}
		engID := resolveEngID(database, engName)

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
		preset := args[0]
		if preset != "ptes" {
			return fmt.Errorf("unknown template: %s (available: ptes)", preset)
		}

		database, engName, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.LoadChecklistPreset(preset); err != nil {
				return fmt.Errorf("load checklist (remote): %w", err)
			}
			fmt.Printf("  [remote] Loaded %s checklist\n", preset)
			return nil
		}
		engID := resolveEngID(database, engName)

		count, err := checklist.LoadPTES(database, engID)
		if err != nil {
			return err
		}
		fmt.Printf("  Loaded PTES checklist: %d items\n", count)
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
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.AddChecklistItem(args[0], args[1]); err != nil {
				return fmt.Errorf("add checklist item (remote): %w", err)
			}
			fmt.Printf("  [remote] Added: [%s] %s\n", args[0], args[1])
			return nil
		}
		engID := resolveEngID(database, engName)

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
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid checklist ID: %s", args[0])
		}

		database, _, dbErr := requireDB()
		if dbErr != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			evidenceID, _ := cmd.Flags().GetInt64("evidence")
			if err := client.ToggleChecklist(id, true, evidenceID); err != nil {
				return fmt.Errorf("check item (remote): %w", err)
			}
			fmt.Printf("  [remote] Checked off item #%d\n", id)
			return nil
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
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid checklist ID: %s", args[0])
		}

		database, _, dbErr := requireDB()
		if dbErr != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.ToggleChecklist(id, false, 0); err != nil {
				return fmt.Errorf("uncheck item (remote): %w", err)
			}
			fmt.Printf("  [remote] Unchecked item #%d\n", id)
			return nil
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
