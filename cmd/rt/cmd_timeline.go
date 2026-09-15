package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/remote"
)

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Show evidence timeline",
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
			entries, cerr := client.GetTimeline()
			if cerr != nil {
				return fmt.Errorf("get timeline (remote): %w", cerr)
			}
			if len(entries) == 0 {
				fmt.Println("  No evidence entries yet.")
				return nil
			}
			fmt.Println()
			for _, e := range entries {
				ts := fmt.Sprintf("%v", e["timestamp"])
				if len(ts) > 19 {
					ts = ts[11:19]
				}
				action := fmt.Sprintf("%v", e["action"])
				input := fmt.Sprintf("%v", e["input"])
				if len(input) > 80 {
					input = input[:77] + "..."
				}
				icons := map[string]string{
					"command": "  $", "tag": "  #", "milestone": " **",
					"bookmark": " >>", "note": "  .", "finding": "  !",
				}
				icon := icons[action]
				if icon == "" {
					icon = "  ?"
				}
				fmt.Printf("  %s %s %s", ts, icon, input)
				if tags, ok := e["tags"].([]interface{}); ok && len(tags) > 0 {
					var tagStrs []string
					for _, t := range tags {
						tagStrs = append(tagStrs, fmt.Sprintf("%v", t))
					}
					fmt.Printf(" [%s]", strings.Join(tagStrs, ", "))
				}
				fmt.Println()
			}
			fmt.Printf("\n  %d entries (remote)\n\n", len(entries))
			return nil
		}

		engID := resolveEngID(database, engName)
		limit, _ := cmd.Flags().GetInt("limit")
		milestonesOnly, _ := cmd.Flags().GetBool("milestones")

		entries, err := evidence.Timeline(database, engID, limit)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("  No evidence entries yet.")
			return nil
		}

		fmt.Println()
		for _, e := range entries {
			if milestonesOnly && e.Action != "milestone" {
				continue
			}
			printTimelineEntry(e)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	timelineCmd.Flags().Int("limit", 100, "Maximum entries to show")
	timelineCmd.Flags().Bool("milestones", false, "Show only milestones")
}

func printTimelineEntry(e evidence.Evidence) {
	icons := map[string]string{
		"command":   "  $",
		"tag":       "  #",
		"milestone": " **",
		"bookmark":  " >>",
		"note":      "  .",
		"finding":   "  !",
	}

	icon := icons[e.Action]
	if icon == "" {
		icon = "  ?"
	}

	ts := e.Timestamp
	if len(ts) > 19 {
		ts = ts[11:19]
	}

	priorityColor := ""
	resetColor := "\033[0m"
	switch e.Priority {
	case "critical":
		priorityColor = "\033[1;31m"
	case "high":
		priorityColor = "\033[1;33m"
	case "medium":
		priorityColor = "\033[33m"
	}

	input := e.Input
	if len(input) > 80 {
		input = input[:77] + "..."
	}

	if priorityColor != "" {
		fmt.Printf("  %s %s%s %s%s", ts, priorityColor, icon, input, resetColor)
	} else {
		fmt.Printf("  %s %s %s", ts, icon, input)
	}

	if e.ExitCode != 0 {
		fmt.Printf(" \033[31m(exit %d)\033[0m", e.ExitCode)
	}

	if len(e.Tags) > 0 {
		fmt.Printf(" \033[2m[%s]\033[0m", strings.Join(e.Tags, ", "))
	}

	fmt.Println()

	if e.Action == "command" && e.Output != "" {
		lines := strings.SplitN(e.Output, "\n", 2)
		first := strings.TrimSpace(lines[0])
		if len(first) > 100 {
			first = first[:97] + "..."
		}
		if first != "" {
			fmt.Printf("           \033[2m→ %s\033[0m\n", first)
		}
	}
}
