package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/capture"
	"github.com/user/rt/internal/remote"
)

var execCmd = &cobra.Command{
	Use:   "exec [flags] [--] <command> [args...]",
	Short: "Execute a command and capture evidence",
	Long: `Run a command with output captured as evidence.

Use --cmd for commands with complex quoting:
  rt exec --cmd "netexec smb 10.0.0.1 -u '' -p ''"

Or use -- to separate rt flags from the command:
  rt exec -- nmap -sV 10.0.0.1

In remote-only mode (rt join without local engagement), evidence is sent
directly to the server without local storage.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rawCmd, _ := cmd.Flags().GetString("cmd")
		var command string
		if rawCmd != "" {
			command = rawCmd
		} else if len(args) > 0 {
			command = shellJoinArgs(args)
		} else {
			return fmt.Errorf("provide a command: rt exec --cmd \"<command>\" or rt exec -- <command>")
		}

		database, sessID, engID, err := getActiveSessionDB()
		if err != nil {
			if remote.IsJoined() {
				return execRemoteOnly(command)
			}
			return err
		}

		operator := getOperator()

		fmt.Printf("  Executing: %s\n\n", command)
		return capture.ExecInteractive(database, sessID, engID, operator, command)
	},
}

func execRemoteOnly(command string) error {
	state, err := remote.LoadState()
	if err != nil {
		return fmt.Errorf("not joined to a server and no local engagement")
	}

	client := remote.NewClient(state.ServerURL, state.APIKey)
	if state.Insecure {
		client.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	fmt.Printf("  [remote] Executing: %s\n\n", command)

	cwd, _ := os.Getwd()
	start := time.Now()

	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/c"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	execCmd := exec.Command(shell, flag, command)
	execCmd.Dir = cwd
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	runErr := execCmd.Run()
	duration := time.Since(start)
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	evReq := remote.EvidenceReq{
		SessionID:  state.SessionID,
		Action:     "exec",
		Input:      command,
		Output:     "",
		ExitCode:   exitCode,
		DurationMs: int(duration.Milliseconds()),
		CWD:        cwd,
		Operator:   state.Operator,
	}

	resp, submitErr := client.SubmitEvidence(evReq)
	if submitErr != nil {
		remote.Enqueue(remote.QueueItem{Type: "evidence", Evidence: &evReq})
		qLen := remote.QueueLen()
		fmt.Printf("  [sync] Server unreachable — evidence queued (%d pending)\n", qLen)
		return nil
	}
	fmt.Printf("  [sync] Evidence #%d synced to server\n", resp.ID)

	if sent, _ := remote.DrainQueue(client); sent > 0 {
		fmt.Printf("  [sync] Drained %d queued item(s)\n", sent)
	}

	return nil
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
