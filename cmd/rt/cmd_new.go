package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/agentctx"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/db"
	"github.com/user/rt/internal/engagement"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new engagement",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		client, _ := cmd.Flags().GetString("client")
		scopeFlag, _ := cmd.Flags().GetString("scope")

		// Check if already exists
		encPath := config.EncryptedDBPath(name)
		if _, err := os.Stat(encPath); err == nil {
			return fmt.Errorf("engagement %q already exists", name)
		}

		// Passphrase
		pass1, err := readPassphrase("Enter passphrase: ")
		if err != nil {
			return fmt.Errorf("read passphrase: %w", err)
		}
		if len(pass1) < 8 {
			return fmt.Errorf("passphrase must be at least 8 characters")
		}

		pass2, err := readPassphrase("Confirm passphrase: ")
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}

		if string(pass1) != string(pass2) {
			return fmt.Errorf("passphrases do not match")
		}

		// Create database
		fmt.Print("Creating encrypted database...")
		if err := db.CreateNew(name, pass1); err != nil {
			return err
		}

		// Open and insert engagement record
		database, err := db.Unlock(name, pass1)
		if err != nil {
			return err
		}

		id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if err := engagement.Create(database, id, name, client); err != nil {
			return err
		}

		config.SetActiveEngagement(name)

		// Lock after creation — user needs to 'rt unlock' or 'rt start' to use
		db.Close()
		if err := db.Lock(name); err != nil {
			fmt.Printf("\n  Warning: could not lock database: %v\n", err)
		}

		// Generate agent context (CLAUDE.md + skills)
		if err := agentctx.GenerateContext(name, client, scopeFlag); err != nil {
			fmt.Printf("\n  Warning: could not generate agent context: %v\n", err)
		}

		fmt.Printf("\n  Engagement %q created.\n", name)
		fmt.Printf("  Database: %s (encrypted)\n", encPath)
		fmt.Printf("  Agent context: %s\n", agentctx.EngagementDir(name))
		fmt.Printf("  Active engagement set to: %s\n\n", name)
		fmt.Println("  Quick start:")
		fmt.Println("    rt start        # begin capturing")
		fmt.Println("    <hack normally>  # RT captures everything")
		fmt.Println("    rt stop          # stop and save")
		return nil
	},
}

func init() {
	newCmd.Flags().String("client", "", "Client name")
	newCmd.Flags().String("scope", "", "Scope CIDRs/domains (comma-separated)")
}
