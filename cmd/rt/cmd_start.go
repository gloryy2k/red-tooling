package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/capture"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/session"
)

var startCmd = &cobra.Command{
	Use:   "start [session-name]",
	Short: "Start a capture session",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		operator, _ := cmd.Flags().GetString("operator")
		if operator == "" {
			operator = getOperator()
		}

		// Check for existing active session
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		existing, _ := session.GetActive(database, engID)
		if existing != nil {
			return fmt.Errorf("session %q is already active. Run 'rt stop' first", existing.Name)
		}

		// Session name
		sessName := fmt.Sprintf("session-%s", time.Now().Format("20060102-150405"))
		if len(args) > 0 {
			sessName = args[0]
		}

		sessID, _ := crypto.RandomHex(8)
		if err := session.Create(database, sessID, engID, sessName, "human", ""); err != nil {
			return err
		}
		session.SetActiveSession(sessID)

		fmt.Printf("\n  Session %q started.\n", sessName)
		fmt.Printf("  Operator: %s\n", operator)
		fmt.Println("  Recording... type commands normally.")
		fmt.Println("  Use 'rt tag/milestone/bookmark/note' to annotate.")
		fmt.Println("  Type 'exit' or run 'rt stop' to end session.")
		fmt.Println()

		// Interactive capture loop
		return runInteractiveCapture(database, sessID, engID, operator)
	},
}

func init() {
	startCmd.Flags().String("operator", "", "Operator name")
}

func runInteractiveCapture(database *sql.DB, sessID, engID, operator string) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	return runLineCapture(database, sessID, engID, operator, sigCh)
}

func runLineCapture(database *sql.DB, sessID, engID, operator string, sigCh chan os.Signal) error {
	scanner := newLineScanner()
	cwd, _ := os.Getwd()
	prompt := fmt.Sprintf("\033[1;32mrt>\033[0m ")

	for {
		fmt.Print(prompt)

		select {
		case <-sigCh:
			fmt.Println("\nSession interrupted.")
			return nil
		default:
		}

		line, ok := scanner.ReadLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Exit commands
		if line == "exit" || line == "quit" {
			fmt.Println("Session ended.")
			return nil
		}

		// Intercept rt subcommands within session
		if strings.HasPrefix(line, "rt ") {
			subCmd := strings.TrimPrefix(line, "rt ")
			if handleInSessionCommand(database, sessID, operator, subCmd) {
				continue
			}
		}

		// Execute and capture
		ev, flagResult, err := capture.RunCommand(database, sessID, engID, operator, line, cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [rt] capture error: %v\n", err)
			continue
		}
		_ = ev
		evidence.PrintAutoFlagResult(flagResult)

		count := evidence.CountBySession(database, sessID)
		fmt.Printf("  \033[2m[rt: %d entries captured]\033[0m\n", count)
	}

	return nil
}

func handleInSessionCommand(database *sql.DB, sessID, operator, subCmd string) bool {
	parts := strings.Fields(subCmd)
	if len(parts) == 0 {
		return false
	}

	cwd, _ := os.Getwd()
	switch parts[0] {
	case "tag":
		if len(parts) > 1 {
			msg := strings.Join(parts[1:], " ")
			evidence.Insert(database, sessID, "tag", msg, "", 0, 0, cwd, nil, "", operator)
			fmt.Printf("  Tagged: %s\n", msg)
		}
		return true
	case "milestone":
		if len(parts) > 1 {
			msg := strings.Join(parts[1:], " ")
			evidence.Insert(database, sessID, "milestone", msg, "", 0, 0, cwd, nil, "high", operator)
			fmt.Printf("  Milestone: %s\n", msg)
		}
		return true
	case "bookmark":
		if len(parts) > 1 {
			msg := strings.Join(parts[1:], " ")
			evidence.Insert(database, sessID, "bookmark", msg, "", 0, 0, cwd, nil, "", operator)
			fmt.Printf("  Bookmarked: %s\n", msg)
		}
		return true
	case "note":
		if len(parts) > 1 {
			msg := strings.Join(parts[1:], " ")
			evidence.Insert(database, sessID, "note", msg, "", 0, 0, cwd, nil, "", operator)
			fmt.Printf("  Note: %s\n", msg)
		}
		return true
	case "stop":
		return false // let caller handle exit
	}
	return false
}
