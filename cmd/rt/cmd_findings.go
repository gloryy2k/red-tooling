package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/findings"
)

var findingCmd = &cobra.Command{
	Use:   "finding <title>",
	Short: "Create a new finding",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		operator := getOperator()

		title := strings.Join(args, " ")
		desc, _ := cmd.Flags().GetString("desc")
		priority, _ := cmd.Flags().GetString("priority")
		evFlag, _ := cmd.Flags().GetString("evidence")
		mitreFlag, _ := cmd.Flags().GetString("mitre")

		var evIDs []int64
		if evFlag != "" {
			for _, s := range strings.Split(evFlag, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid evidence ID: %s", s)
				}
				evIDs = append(evIDs, id)
			}
		}

		var mitre []string
		if mitreFlag != "" {
			for _, m := range strings.Split(mitreFlag, ",") {
				mitre = append(mitre, strings.TrimSpace(m))
			}
		}

		if priority == "" {
			priority = "medium"
		}

		f, err := findings.Create(database, engID, title, desc, priority, operator, evIDs, mitre)
		if err != nil {
			return err
		}

		fmt.Printf("  Finding #%d created: %s [%s]\n", f.ID, f.Title, f.Priority)
		return nil
	},
}

var findingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "List findings",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		list, err := findings.List(database, engID)
		if err != nil {
			return err
		}

		if len(list) == 0 {
			fmt.Println("  No findings.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-4s %-10s %-12s %-50s %s\n", "ID", "PRIORITY", "STATUS", "TITLE", "EVIDENCE")
		fmt.Printf("  %-4s %-10s %-12s %-50s %s\n",
			strings.Repeat("-", 4), strings.Repeat("-", 10), strings.Repeat("-", 12),
			strings.Repeat("-", 50), strings.Repeat("-", 10))

		for _, f := range list {
			evStr := fmt.Sprintf("%d", len(f.EvidenceIDs))
			title := f.Title
			if len(title) > 48 {
				title = title[:45] + "..."
			}

			statusColor := ""
			resetColor := "\033[0m"
			switch f.Verified {
			case "confirmed":
				statusColor = "\033[32m"
			case "false-positive":
				statusColor = "\033[90m"
			default:
				statusColor = "\033[33m"
			}

			priColor := priorityColor(f.Priority)
			fmt.Printf("  %-4d %s%-10s\033[0m %s%-12s%s %-50s %s\n",
				f.ID, priColor, f.Priority, statusColor, f.Verified, resetColor, title, evStr)
		}

		uv, cf, fp := findings.CountByStatus(database, engID)
		fmt.Printf("\n  %d finding(s): %d unverified, %d confirmed, %d false-positive\n\n", uv+cf+fp, uv, cf, fp)
		return nil
	},
}

var verifyFindingCmd = &cobra.Command{
	Use:   "verify-finding <id> <confirmed|false-positive>",
	Short: "Mark a finding as confirmed or false-positive",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid finding ID: %s", args[0])
		}

		status := args[1]
		if status != "confirmed" && status != "false-positive" {
			return fmt.Errorf("status must be 'confirmed' or 'false-positive'")
		}

		note, _ := cmd.Flags().GetString("note")
		operator := getOperator()

		if err := findings.Verify(database, id, status, operator, note); err != nil {
			return err
		}

		fmt.Printf("  Finding #%d marked as %s\n", id, status)
		return nil
	},
}

var mergeFindingsCmd = &cobra.Command{
	Use:   "merge-findings <id1,id2,...> <new-title>",
	Short: "Merge multiple findings into one",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		var ids []int64
		for _, s := range strings.Split(args[0], ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid finding ID: %s", s)
			}
			ids = append(ids, id)
		}

		newTitle := strings.Join(args[1:], " ")
		operator := getOperator()

		merged, err := findings.Merge(database, ids, newTitle, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Merged %d findings → Finding #%d: %s [%s]\n", len(ids), merged.ID, merged.Title, merged.Priority)
		return nil
	},
}

var recommendCmd = &cobra.Command{
	Use:   "recommend <finding-id> <recommendation>",
	Short: "Set a recommendation for a finding",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid finding ID: %s", args[0])
		}

		rec := strings.Join(args[1:], " ")
		operator := getOperator()

		if err := findings.SetRecommendation(database, id, rec, operator); err != nil {
			return err
		}

		fmt.Printf("  Recommendation set for Finding #%d\n", id)
		return nil
	},
}

func init() {
	findingCmd.Flags().String("desc", "", "Finding description")
	findingCmd.Flags().String("priority", "medium", "Priority: critical, high, medium, low, info")
	findingCmd.Flags().String("evidence", "", "Comma-separated evidence IDs")
	findingCmd.Flags().String("mitre", "", "Comma-separated MITRE ATT&CK IDs")

	verifyFindingCmd.Flags().String("note", "", "Verification note")
}

func priorityColor(p string) string {
	switch p {
	case "critical":
		return "\033[31m"
	case "high":
		return "\033[91m"
	case "medium":
		return "\033[33m"
	case "low":
		return "\033[36m"
	default:
		return "\033[37m"
	}
}
