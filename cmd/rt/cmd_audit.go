package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/remote"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")

		database, _, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			entries, cerr := client.GetAudit(limit)
			if cerr != nil {
				return fmt.Errorf("get audit (remote): %w", cerr)
			}
			if len(entries) == 0 {
				fmt.Println("  No audit entries.")
				return nil
			}
			fmt.Println()
			fmt.Printf("  %-20s %-12s %-22s %-12s %s\n", "TIME", "OPERATOR", "ACTION", "TARGET", "DETAIL")
			fmt.Printf("  %-20s %-12s %-22s %-12s %s\n",
				strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 22),
				strings.Repeat("-", 12), strings.Repeat("-", 30))
			for _, e := range entries {
				ts := fmt.Sprintf("%v", e["timestamp"])
				if len(ts) > 19 {
					ts = ts[:19]
				}
				op := fmt.Sprintf("%v", e["operator"])
				action := fmt.Sprintf("%v", e["action"])
				target := ""
				if tt, ok := e["target_type"]; ok && tt != nil && fmt.Sprintf("%v", tt) != "" {
					target = fmt.Sprintf("%v", tt)
					if tid, ok := e["target_id"]; ok && tid != nil && fmt.Sprintf("%v", tid) != "" {
						target += ":" + fmt.Sprintf("%v", tid)
					}
				}
				if len(target) > 12 {
					target = target[:9] + "..."
				}
				detail := fmt.Sprintf("%v", e["detail"])
				if len(detail) > 30 {
					detail = detail[:27] + "..."
				}
				fmt.Printf("  %-20s %-12s %-22s %-12s %s\n",
					ts, truncate(op, 12), action, target, detail)
			}
			fmt.Printf("\n  %d entries shown (remote).\n\n", len(entries))
			return nil
		}

		entries, err := audit.List(database, limit)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("  No audit entries.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-20s %-12s %-22s %-12s %s\n", "TIME", "OPERATOR", "ACTION", "TARGET", "DETAIL")
		fmt.Printf("  %-20s %-12s %-22s %-12s %s\n",
			strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 22),
			strings.Repeat("-", 12), strings.Repeat("-", 30))

		for _, e := range entries {
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}

			target := ""
			if e.TargetType != "" {
				target = e.TargetType
				if e.TargetID != "" {
					target += ":" + e.TargetID
				}
			}
			if len(target) > 12 {
				target = target[:9] + "..."
			}

			detail := e.Detail
			if len(detail) > 30 {
				detail = detail[:27] + "..."
			}

			warning := ""
			if isWarningAction(e.Action) {
				warning = " \033[33m!\033[0m"
			}

			fmt.Printf("  %-20s %-12s %-22s %-12s %s%s\n",
				ts, truncate(e.Operator, 12), e.Action, target, detail, warning)
		}
		fmt.Printf("\n  %d entries shown.\n\n", len(entries))
		return nil
	},
}

func init() {
	auditCmd.Flags().Int("limit", 50, "Maximum entries to show")
}

func isWarningAction(action string) bool {
	warnings := []string{"cred.view", "cred.export", "chain.broken", "api.auth_fail", "evidence.delete", "evidence.redact"}
	for _, w := range warnings {
		if action == w {
			return true
		}
	}
	return false
}
