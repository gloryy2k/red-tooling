package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
	"github.com/user/rt/internal/scope"
)

func remoteClientFromState() (*remote.Client, *remote.ConnectionState, error) {
	state, err := remote.LoadState()
	if err != nil {
		return nil, nil, fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
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
	return client, state, nil
}

var scopeCmd = &cobra.Command{
	Use:   "scope <hosts>",
	Short: "Add hosts/CIDRs to engagement scope (comma-separated)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostList := args[0]
		for _, a := range args[1:] {
			hostList += "," + a
		}

		database, engName, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, err := remoteClientFromState()
			if err != nil {
				return err
			}
			resp, err := client.AddScope(hostList)
			if err != nil {
				return fmt.Errorf("add scope (remote): %w", err)
			}
			fmt.Printf("  Added %d hosts to scope (remote)\n", resp.Added)
			return nil
		}

		engID := resolveEngID(database, engName)
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
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, err := remoteClientFromState()
			if err != nil {
				return err
			}
			result, err := client.GetScope()
			if err != nil {
				return fmt.Errorf("list scope (remote): %w", err)
			}

			hosts, _ := result["hosts"].([]interface{})
			if len(hosts) == 0 {
				fmt.Println("  No scope hosts defined")
				return nil
			}

			total := toInt(result["total"])
			tested := toInt(result["tested"])
			fmt.Printf("  Scope: %d/%d tested (remote)\n\n", tested, total)

			for _, h := range hosts {
				if hm, ok := h.(map[string]interface{}); ok {
					status := "[ ]"
					if tb, ok := hm["Tested"].(bool); ok && tb {
						status = "[x]"
					}
					host := hm["Host"]
					if host == nil {
						host = hm["host"]
					}
					fmt.Printf("  %s %s\n", status, host)
				}
			}
			return nil
		}

		engID := resolveEngID(database, engName)
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
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, state, err := remoteClientFromState()
			if err != nil {
				return err
			}
			if err := client.MarkScopeTested(args[0], state.SessionID); err != nil {
				return fmt.Errorf("mark tested (remote): %w", err)
			}
			fmt.Printf("  Marked %s as tested (remote)\n", args[0])
			return nil
		}

		engID := resolveEngID(database, engName)
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
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, err := remoteClientFromState()
			if err != nil {
				return err
			}
			result, err := client.GetScope()
			if err != nil {
				return fmt.Errorf("list scope (remote): %w", err)
			}

			hosts, _ := result["hosts"].([]interface{})
			var untested []string
			for _, h := range hosts {
				if hm, ok := h.(map[string]interface{}); ok {
					tested, _ := hm["Tested"].(bool)
					if !tested {
						host := hm["Host"]
						if host == nil {
							host = hm["host"]
						}
						if host != nil {
							untested = append(untested, fmt.Sprintf("%v", host))
						}
					}
				}
			}

			if len(untested) == 0 {
				fmt.Println("  All scope hosts have been tested!")
				return nil
			}
			fmt.Printf("  %d untested hosts (remote):\n\n", len(untested))
			for _, h := range untested {
				fmt.Printf("  - %s\n", h)
			}
			return nil
		}

		engID := resolveEngID(database, engName)
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
