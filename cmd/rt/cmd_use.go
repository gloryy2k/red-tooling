package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
)

var useCmd = &cobra.Command{
	Use:   "use <engagement>",
	Short: "Switch active engagement",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !engagementExists(name) {
			return fmt.Errorf("engagement %q not found. Run 'rt ls' to see available engagements", name)
		}
		config.SetActiveEngagement(name)
		fmt.Printf("Active engagement: %s\n", name)
		return nil
	},
}
