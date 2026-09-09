package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/evidence"
)

var verifyChainCmd = &cobra.Command{
	Use:   "verify-chain",
	Short: "Verify evidence hash chain integrity",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		results, err := evidence.VerifyChain(database, engID)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("  No sessions to verify.")
			return nil
		}

		fmt.Println()
		allGood := true
		for _, r := range results {
			if r.Intact {
				fmt.Printf("  \033[32m+\033[0m Session %s: %d entries, chain intact\n", r.SessionName, r.EntryCount)
			} else {
				fmt.Printf("  \033[31m!\033[0m Session %s: %d entries, BROKEN at entry #%d\n", r.SessionName, r.EntryCount, r.BrokenAt)
				if r.Detail != "" {
					fmt.Printf("    %s\n", r.Detail)
				}
				allGood = false
			}
		}

		fmt.Println()
		if allGood {
			fmt.Println("  All chains intact.")
		} else {
			fmt.Println("  WARNING: One or more chains have integrity violations.")
			fmt.Println("  Evidence may have been tampered with.")
		}
		fmt.Println()
		return nil
	},
}
