package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/scope"
)

var scopeCmd = &cobra.Command{
	Use:   "scope <hosts>",
	Short: "Add hosts/CIDRs to engagement scope (comma-separated)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		hostList := args[0]
		for _, a := range args[1:] {
			hostList += "," + a
		}

		operator := getOperator()
		count, err := scope.Add(database, engID, hostList, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Added %d hosts to scope\n", count)
		return nil
	},
}

var scopeListCmd = &cobra.Command{
	Use:   "scope-list",
	Short: "List all in-scope hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		hosts, err := scope.List(database, engID)
		if err != nil {
			return err
		}

		if len(hosts) == 0 {
			fmt.Println("  No scope hosts defined")
			return nil
		}

		total, tested := scope.Stats(database, engID)
		fmt.Printf("  Scope: %d/%d tested\n\n", tested, total)

		for _, h := range hosts {
			status := "[ ]"
			if h.Tested {
				status = "[x]"
			}
			fmt.Printf("  %s %s", status, h.Host)
			if h.TestedAt != "" {
				fmt.Printf("  (tested %s)", h.TestedAt[:10])
			}
			fmt.Println()
		}
		return nil
	},
}

var scopeTestedCmd = &cobra.Command{
	Use:   "scope-tested <host>",
	Short: "Mark a scope host as tested",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		operator := getOperator()
		sessionID, _ := cmd.Flags().GetString("session")

		if err := scope.MarkTested(database, engID, args[0], sessionID, operator); err != nil {
			return err
		}

		fmt.Printf("  Marked %s as tested\n", args[0])
		return nil
	},
}

var scopeUntestedCmd = &cobra.Command{
	Use:   "scope-untested",
	Short: "List untested scope hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		hosts, err := scope.Untested(database, engID)
		if err != nil {
			return err
		}

		if len(hosts) == 0 {
			fmt.Println("  All scope hosts have been tested!")
			return nil
		}

		fmt.Printf("  %d untested hosts:\n\n", len(hosts))
		for _, h := range hosts {
			fmt.Printf("  - %s\n", h.Host)
		}
		return nil
	},
}

func init() {
	scopeTestedCmd.Flags().String("session", "", "Session ID that tested this host")
}
