package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join a remote RT server as an agent",
	Long: `Connect to a central RT server and start syncing evidence.
After joining, every 'rt exec' will automatically submit evidence to the server.
Use 'rt leave' to disconnect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		apiKey, _ := cmd.Flags().GetString("key")
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

		if remote.IsJoined() {
			state, _ := remote.LoadState()
			if state != nil {
				return fmt.Errorf("already joined server %s (session: %s). Run 'rt leave' first", state.ServerURL, state.SessionID)
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
			return fmt.Errorf("cannot reach server: %w", err)
		}

		sessionID := "agent-" + uuid.New().String()[:8]
		err := client.StartSession(remote.SessionReq{
			ID:       sessionID,
			Name:     "agent-" + operator,
			Source:   "rt-join",
			Operator: operator,
		})
		if err != nil {
			return fmt.Errorf("start remote session: %w", err)
		}

		state := &remote.ConnectionState{
			ServerURL: serverURL,
			APIKey:    apiKey,
			SessionID: sessionID,
			Operator:  operator,
			Insecure:  insecure,
		}
		if err := remote.SaveState(state); err != nil {
			return fmt.Errorf("save connection state: %w", err)
		}

		fmt.Printf("  Joined server: %s\n", serverURL)
		fmt.Printf("  Session: %s\n", sessionID)
		fmt.Printf("  Operator: %s\n", operator)
		fmt.Printf("\n  All 'rt exec' commands will now auto-sync to the server.\n")
		fmt.Printf("  Run 'rt leave' to disconnect.\n")
		return nil
	},
}

var leaveCmd = &cobra.Command{
	Use:   "leave",
	Short: "Leave the remote RT server",
	Long:  "Disconnect from the central server and stop auto-syncing evidence.",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := remote.LoadState()
		if err != nil {
			return fmt.Errorf("not currently joined to any server")
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

		if err := client.StopSession(state.SessionID, state.Operator); err != nil {
			fmt.Printf("  Warning: could not stop remote session: %v\n", err)
		}

		if err := remote.ClearState(); err != nil {
			return fmt.Errorf("clear connection state: %w", err)
		}

		fmt.Printf("  Left server: %s\n", state.ServerURL)
		fmt.Printf("  Session %s stopped.\n", state.SessionID)
		return nil
	},
}

func init() {
	joinCmd.Flags().String("server", "", "Central RT server URL (or RT_SERVER env)")
	joinCmd.Flags().String("key", "", "API key (or RT_API_KEY env)")
	joinCmd.Flags().String("operator", "", "Operator name")
	joinCmd.Flags().Bool("insecure", false, "Skip TLS verification")
}
