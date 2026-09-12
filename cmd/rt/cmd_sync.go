package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch engagement context from remote server",
	Long: `Pull scope, findings, and checklist status from the central RT server.
Requires an active connection (rt join) or --server/--key flags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		apiKey, _ := cmd.Flags().GetString("key")
		insecure, _ := cmd.Flags().GetBool("insecure")

		if serverURL == "" || apiKey == "" {
			state, err := remote.LoadState()
			if err != nil {
				return fmt.Errorf("not joined to a server — use 'rt join' or pass --server/--key")
			}
			if serverURL == "" {
				serverURL = state.ServerURL
			}
			if apiKey == "" {
				apiKey = state.APIKey
			}
			insecure = insecure || state.Insecure
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

		ctx, err := client.GetContext()
		if err != nil {
			return fmt.Errorf("fetch context: %w", err)
		}

		printContext(ctx)
		return nil
	},
}

func printContext(ctx *remote.ContextResp) {
	fmt.Println()

	// Engagement info
	if eng := ctx.Engagement; eng != nil {
		fmt.Printf("  Engagement: %v\n", eng["name"])
		if c, ok := eng["client"]; ok && c != "" {
			fmt.Printf("  Client:     %v\n", c)
		}
		fmt.Printf("  Status:     %v\n", eng["status"])
		if sd, ok := eng["start_date"]; ok && sd != "" {
			fmt.Printf("  Start:      %v\n", sd)
		}
		if ed, ok := eng["end_date"]; ok && ed != "" {
			fmt.Printf("  End:        %v\n", ed)
		}
	}

	// Scope
	if s := ctx.Scope; s != nil {
		total := toInt(s["total"])
		tested := toInt(s["tested"])
		fmt.Printf("\n  Scope: %d hosts (%d tested, %d remaining)\n", total, tested, total-tested)
		if hosts, ok := s["hosts"].([]interface{}); ok {
			for _, h := range hosts {
				if hm, ok := h.(map[string]interface{}); ok {
					mark := " "
					if tb, ok := hm["Tested"].(bool); ok && tb {
						mark = "x"
					}
					host := hm["Host"]
					if host == nil {
						host = hm["host"]
					}
					fmt.Printf("    [%s] %v\n", mark, host)
				}
			}
		}
	}

	// Findings
	if len(ctx.Findings) > 0 {
		fmt.Printf("\n  Findings: %d total\n", len(ctx.Findings))
		for _, f := range ctx.Findings {
			pri := f["priority"]
			if pri == nil || pri == "" {
				pri = "info"
			}
			fmt.Printf("    #%-4v [%-8v] %v (%v)\n", f["id"], pri, f["title"], f["status"])
		}
	} else {
		fmt.Println("\n  Findings: none")
	}

	// Checklist
	if cl := ctx.Checklist; cl != nil {
		total := toInt(cl["total"])
		done := toInt(cl["done"])
		if total > 0 {
			pct := 0
			if total > 0 {
				pct = done * 100 / total
			}
			fmt.Printf("\n  Checklist: %d/%d complete (%d%%)\n", done, total, pct)
			if items, ok := cl["items"].([]interface{}); ok {
				for _, it := range items {
					if im, ok := it.(map[string]interface{}); ok {
						mark := " "
						if cb, ok := im["checked"].(bool); ok && cb {
							mark = "x"
						}
						fmt.Printf("    [%s] [%v] %v\n", mark, im["category"], im["item"])
					}
				}
			}
		} else {
			fmt.Println("\n  Checklist: not loaded")
		}
	}

	fmt.Println()
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func init() {
	syncCmd.Flags().String("server", "", "Server URL (uses joined server if omitted)")
	syncCmd.Flags().String("key", "", "API key (uses joined key if omitted)")
	syncCmd.Flags().Bool("insecure", false, "Skip TLS verification")
}
