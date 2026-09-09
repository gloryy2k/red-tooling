package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/capture"
)

var execCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute a command and capture evidence",
	Long:  "Run a command with output captured as evidence. Use when not in an active session.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, sessID, engID, err := getActiveSessionDB()
		if err != nil {
			return err
		}

		command := strings.Join(args, " ")
		operator := getOperator()

		fmt.Printf("  Executing: %s\n\n", command)
		return capture.ExecInteractive(database, sessID, engID, operator, command)
	},
}
