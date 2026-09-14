package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
)

func buildRemoteClient(cmd *cobra.Command) (*remote.Client, error) {
	serverURL, _ := cmd.Flags().GetString("server")
	apiKey, _ := cmd.Flags().GetString("key")
	insecure, _ := cmd.Flags().GetBool("insecure")
	timeout, _ := cmd.Flags().GetInt("timeout")

	if serverURL == "" {
		serverURL = os.Getenv("RT_SERVER")
	}
	if apiKey == "" {
		apiKey = os.Getenv("RT_API_KEY")
	}
	if serverURL == "" || apiKey == "" {
		return nil, fmt.Errorf("--server and --key required (or set RT_SERVER / RT_API_KEY)")
	}

	if !insecure {
		if v := os.Getenv("RT_INSECURE"); v == "1" || v == "true" {
			insecure = true
		}
	}

	if timeout == 0 {
		if v := os.Getenv("RT_TIMEOUT"); v != "" {
			if t, err := strconv.Atoi(v); err == nil {
				timeout = t
			}
		}
		if timeout == 0 {
			timeout = 10
		}
	}

	client := remote.NewClient(serverURL, apiKey)
	client.HTTPClient.Timeout = time.Duration(timeout) * time.Second

	if insecure {
		client.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client, nil
}

var remoteExecCmd = &cobra.Command{
	Use:   "remote-exec [flags] [--] <command> [args...]",
	Short: "[DEPRECATED: use 'rt join' + 'rt exec'] Execute a command and submit evidence to server",
	Long: `DEPRECATED: Use 'rt join' then 'rt exec' instead — it's simpler and supports offline queueing.

Run a command locally and POST the result to a remote RT server.
Requires --server and --key flags (or RT_SERVER / RT_API_KEY env vars).
Set RT_INSECURE=1 to skip TLS verification for self-signed certs.
Set RT_TIMEOUT=N to set connection timeout in seconds (default: 10).

Use --cmd for commands with complex quoting:
  rt remote-exec --cmd "netexec smb 10.0.0.1 -u '' -p ''"`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  [DEPRECATED] Use 'rt join' + 'rt exec' instead — simpler and supports offline queueing.")
		fmt.Println()
		client, err := buildRemoteClient(cmd)
		if err != nil {
			return err
		}

		sessionID, _ := cmd.Flags().GetString("session")
		operator, _ := cmd.Flags().GetString("operator")

		if operator == "" {
			operator = os.Getenv("USER")
			if operator == "" {
				operator = os.Getenv("USERNAME")
			}
		}

		if err := client.Ping(); err != nil {
			return fmt.Errorf("server unreachable at %s — check firewall/VPN: %w", client.BaseURL, err)
		}

		if sessionID == "" {
			sessionID = "remote-" + uuid.New().String()[:8]
			err := client.StartSession(remote.SessionReq{
				ID:       sessionID,
				Name:     "remote-agent-" + operator,
				Source:   "remote-exec",
				Operator: operator,
			})
			if err != nil {
				return fmt.Errorf("start session: %w", err)
			}
			fmt.Printf("  [session] Started: %s\n", sessionID)
		}

		rawCmd, _ := cmd.Flags().GetString("cmd")
		var command string
		if rawCmd != "" {
			command = rawCmd
		} else if len(args) > 0 {
			command = shellJoinArgs(args)
		} else {
			return fmt.Errorf("provide a command: rt remote-exec --cmd \"<command>\" or rt remote-exec -- <command>")
		}
		fmt.Printf("  [exec] %s\n", command)

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
		output, err := execCmd.CombinedOutput()
		duration := time.Since(start)
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		outStr := string(output)
		if outStr != "" {
			fmt.Printf("  [output]\n%s\n", outStr)
		}

		resp, err := client.SubmitEvidence(remote.EvidenceReq{
			SessionID:  sessionID,
			Action:     "exec",
			Input:      command,
			Output:     outStr,
			ExitCode:   exitCode,
			DurationMs: int(duration.Milliseconds()),
			CWD:        cwd,
			Operator:   operator,
		})
		if err != nil {
			return fmt.Errorf("submit evidence: %w", err)
		}

		fmt.Printf("  [evidence] #%d submitted (hash: %s)\n", resp.ID, resp.Hash[:16]+"...")
		return nil
	},
}

var remoteSessionCmd = &cobra.Command{
	Use:   "remote-session <start|stop>",
	Short: "[DEPRECATED: use 'rt join'/'rt leave'] Manage a remote session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  [DEPRECATED] Use 'rt join'/'rt leave' instead.")
		fmt.Println()
		client, err := buildRemoteClient(cmd)
		if err != nil {
			return err
		}

		sessionID, _ := cmd.Flags().GetString("session")
		sessionName, _ := cmd.Flags().GetString("name")
		operator, _ := cmd.Flags().GetString("operator")

		if operator == "" {
			operator = os.Getenv("USER")
			if operator == "" {
				operator = os.Getenv("USERNAME")
			}
		}

		switch args[0] {
		case "start":
			if sessionID == "" {
				sessionID = "remote-" + uuid.New().String()[:8]
			}
			if sessionName == "" {
				sessionName = "remote-" + operator
			}
			err := client.StartSession(remote.SessionReq{
				ID:       sessionID,
				Name:     sessionName,
				Source:   "remote",
				Operator: operator,
			})
			if err != nil {
				return fmt.Errorf("server unreachable at %s — check firewall/VPN: %w", client.BaseURL, err)
			}
			fmt.Printf("  Session started: %s\n", sessionID)
			fmt.Printf("  Use this ID with: rt remote-exec --session %s ...\n", sessionID)

		case "stop":
			if sessionID == "" {
				return fmt.Errorf("--session required for stop")
			}
			if err := client.StopSession(sessionID, operator); err != nil {
				return err
			}
			fmt.Printf("  Session stopped: %s\n", sessionID)

		default:
			return fmt.Errorf("action must be 'start' or 'stop'")
		}

		return nil
	},
}

func init() {
	for _, cmd := range []*cobra.Command{remoteExecCmd, remoteSessionCmd} {
		cmd.Flags().String("server", "", "Central RT server URL (or RT_SERVER env)")
		cmd.Flags().String("key", "", "API key for authentication (or RT_API_KEY env)")
		cmd.Flags().String("session", "", "Session ID to use")
		cmd.Flags().String("operator", "", "Operator name")
		cmd.Flags().Bool("insecure", false, "Skip TLS verification (or RT_INSECURE=1)")
		cmd.Flags().Int("timeout", 0, "Connection timeout in seconds (default 10, or RT_TIMEOUT)")
	}
	remoteSessionCmd.Flags().String("name", "", "Session name (for start)")
	remoteExecCmd.Flags().String("cmd", "", "Raw command string (preserves quoting exactly)")
}
