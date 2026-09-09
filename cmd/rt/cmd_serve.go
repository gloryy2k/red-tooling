package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	operatorPkg "github.com/user/rt/internal/operator"
	"github.com/user/rt/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		listen, _ := cmd.Flags().GetString("listen")
		tlsCert, _ := cmd.Flags().GetString("tls-cert")
		tlsKey, _ := cmd.Flags().GetString("tls-key")

		srv := server.New(database, engID, engName, listen, tlsCert, tlsKey)
		return srv.Start()
	},
}

func init() {
	serveCmd.Flags().String("listen", "localhost:8080", "Listen address (host:port)")
	serveCmd.Flags().String("tls-cert", "", "TLS certificate file")
	serveCmd.Flags().String("tls-key", "", "TLS private key file")
}

var operatorAddCmd = &cobra.Command{
	Use:   "operator-add <name> --role <role>",
	Short: "Add an operator to the engagement",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		name := args[0]
		role, _ := cmd.Flags().GetString("role")
		if role == "" {
			role = "operator"
		}
		creator := getOperator()

		op, err := operatorPkg.Add(database, engID, name, role, creator)
		if err != nil {
			return err
		}

		fmt.Printf("  Operator added: %s [%s]\n", op.ID, op.Role)
		fmt.Printf("  API Key: %s\n", op.APIKey)
		fmt.Printf("  (Save this key — it cannot be shown again)\n")
		return nil
	},
}

var operatorListCmd = &cobra.Command{
	Use:   "operator-list",
	Short: "List operators",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		ops, err := operatorPkg.List(database, engID)
		if err != nil {
			return err
		}

		if len(ops) == 0 {
			fmt.Println("  No operators. Use 'rt operator-add <name> --role <role>' to add.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-20s %-12s %-8s %-20s\n", "OPERATOR", "ROLE", "HAS KEY", "LAST SEEN")
		fmt.Printf("  %-20s %-12s %-8s %-20s\n",
			strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 8), strings.Repeat("-", 20))
		for _, o := range ops {
			hasKey := "no"
			if o.APIKeyHash != "" {
				hasKey = "yes"
			}
			lastSeen := o.LastSeenAt
			if lastSeen == "" {
				lastSeen = "never"
			}
			fmt.Printf("  %-20s %-12s %-8s %-20s\n", o.ID, o.Role, hasKey, lastSeen)
		}
		fmt.Printf("\n  %d operator(s)\n\n", len(ops))
		return nil
	},
}

var operatorRotateCmd = &cobra.Command{
	Use:   "operator-rotate <name>",
	Short: "Rotate an operator's API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		name := args[0]
		creator := getOperator()

		newKey, err := operatorPkg.RotateKey(database, engID, name, creator)
		if err != nil {
			return err
		}

		fmt.Printf("  API key rotated for: %s\n", name)
		fmt.Printf("  New API Key: %s\n", newKey)
		fmt.Printf("  (Previous key is now invalid)\n")
		return nil
	},
}

func init() {
	operatorAddCmd.Flags().String("role", "operator", "Role: lead, operator, reviewer, viewer")
}
