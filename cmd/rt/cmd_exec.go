package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/capture"
)

var execCmd = &cobra.Command{
	Use:   "exec [flags] [--] <command> [args...]",
	Short: "Execute a command and capture evidence",
	Long: `Run a command with output captured as evidence.

Use --cmd for commands with complex quoting:
  rt exec --cmd "netexec smb 10.0.0.1 -u '' -p ''"

Or use -- to separate rt flags from the command:
  rt exec -- nmap -sV 10.0.0.1`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		database, sessID, engID, err := getActiveSessionDB()
		if err != nil {
			return err
		}

		rawCmd, _ := cmd.Flags().GetString("cmd")
		var command string
		if rawCmd != "" {
			command = rawCmd
		} else if len(args) > 0 {
			command = shellJoinArgs(args)
		} else {
			return fmt.Errorf("provide a command: rt exec --cmd \"<command>\" or rt exec -- <command>")
		}

		operator := getOperator()

		fmt.Printf("  Executing: %s\n\n", command)
		return capture.ExecInteractive(database, sessID, engID, operator, command)
	},
}

func shellJoinArgs(args []string) string {
	var parts []string
	for _, a := range args {
		if a == "" {
			parts = append(parts, "''")
		} else if strings.ContainsAny(a, " \t\n\"'\\$`!#&|;(){}[]<>?*~") {
			escaped := strings.ReplaceAll(a, "'", "'\\''")
			parts = append(parts, "'"+escaped+"'")
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

func init() {
	execCmd.Flags().String("cmd", "", "Raw command string (preserves quoting exactly)")
}
