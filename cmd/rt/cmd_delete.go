package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/evidence"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <evidence-id> [evidence-id...]",
	Short: "Soft-delete evidence entries",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		var ids []int64
		for _, s := range args {
			for _, part := range strings.Split(s, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid evidence ID: %s", part)
				}
				ids = append(ids, id)
			}
		}

		operator := getOperator()
		if err := evidence.SoftDelete(database, ids, operator); err != nil {
			return err
		}

		fmt.Printf("  Soft-deleted %d evidence entries (audit-logged)\n", len(ids))
		return nil
	},
}

var redactCmd = &cobra.Command{
	Use:   "redact <evidence-id> <reason>",
	Short: "Redact evidence output (replace with reason marker)",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid evidence ID: %s", args[0])
		}

		reason := strings.Join(args[1:], " ")
		operator := getOperator()

		if err := evidence.Redact(database, id, reason, operator); err != nil {
			return err
		}

		fmt.Printf("  Evidence #%d redacted: [REDACTED: %s] (audit-logged)\n", id, reason)
		return nil
	},
}
