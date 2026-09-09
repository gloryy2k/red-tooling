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

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
)

var remoteExecCmd = &cobra.Command{
	Use:   "remote-exec <command>",
	Short: "Execute a command and submit evidence to a central RT server",
	Long: `Run a command locally and POST the result to a remote RT server.
Requires --server and --key flags (or RT_SERVER / RT_API_KEY env vars).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		apiKey, _ := cmd.Flags().GetString("key")
		sessionID, _ := cmd.Flags().GetString("session")
		operator, _ := cmd.Flags().GetString("operator")
		insecure, _ := cmd.Flags().GetBool("insecure")

		if serverURL == "" {
			serverURL = os.Getenv("RT_SERVER")
		}
		if apiKey == "" {
			apiKey = os.Getenv("RT_API_KEY")
		}
		if serverURL == "" || apiKey == "" {
			return fmt.Errorf("--server and --key required (or set RT_SERVER / RT_API_KEY)")
		}

		if operator == "" {
			operator = os.Getenv("USER")
			if operator == "" {
				operator = os.Getenv("USERNAME")
			}
		}

		client := remote.NewClient(serverURL, apiKey)
		if insecure {
			client.HTTPClient = &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
		}

		if err := client.Ping(); err != nil {
			return fmt.Errorf("server connection failed: %w", err)
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

		command := strings.Join(args, " ")
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
		if len(outStr) > 500 {
			fmt.Printf("  [output] %s...\n", outStr[:500])
		} else if outStr != "" {
			fmt.Printf("  [output] %s\n", outStr)
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
	Short: "Manage a remote session on the central server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		apiKey, _ := cmd.Flags().GetString("key")
		sessionID, _ := cmd.Flags().GetString("session")
		sessionName, _ := cmd.Flags().GetString("name")
		operator, _ := cmd.Flags().GetString("operator")
		insecure, _ := cmd.Flags().GetBool("insecure")

		if serverURL == "" {
			serverURL = os.Getenv("RT_SERVER")
		}
		if apiKey == "" {
			apiKey = os.Getenv("RT_API_KEY")
		}
		if serverURL == "" || apiKey == "" {
			return fmt.Errorf("--server and --key required (or set RT_SERVER / RT_API_KEY)")
		}

		if operator == "" {
			operator = os.Getenv("USER")
			if operator == "" {
				operator = os.Getenv("USERNAME")
			}
		}

		client := remote.NewClient(serverURL, apiKey)
		if insecure {
			client.HTTPClient = &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
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
				return err
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
		cmd.Flags().Bool("insecure", false, "Skip TLS verification (self-signed certs)")
	}
	remoteSessionCmd.Flags().String("name", "", "Session name (for start)")
}
