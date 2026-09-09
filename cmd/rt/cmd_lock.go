package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/db"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock (re-encrypt) the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		engName := config.GetActiveEngagement()
		if engName == "" {
			return fmt.Errorf("no active engagement")
		}

		if !db.IsUnlocked(engName) {
			fmt.Println("Database already locked.")
			return nil
		}

		db.Close()
		if err := db.Lock(engName); err != nil {
			return err
		}

		fmt.Printf("Database locked: %s\n", engName)
		return nil
	},
}
