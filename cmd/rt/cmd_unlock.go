package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/db"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the encrypted database",
	RunE: func(cmd *cobra.Command, args []string) error {
		engName := config.GetActiveEngagement()
		if engName == "" {
			return fmt.Errorf("no active engagement. Run 'rt use' first")
		}

		if db.IsUnlocked(engName) {
			fmt.Println("Database already unlocked.")
			return nil
		}

		pass, err := readPassphrase("Enter passphrase: ")
		if err != nil {
			return err
		}

		if _, err := db.Unlock(engName, pass); err != nil {
			return err
		}
		crypto.ZeroBytes(pass)

		fmt.Printf("Database unlocked: %s\n", engName)
		return nil
	},
}

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change database passphrase",
	RunE: func(cmd *cobra.Command, args []string) error {
		engName := config.GetActiveEngagement()
		if engName == "" {
			return fmt.Errorf("no active engagement")
		}

		oldPass, err := readPassphrase("Current passphrase: ")
		if err != nil {
			return err
		}

		newPass1, err := readPassphrase("New passphrase: ")
		if err != nil {
			return err
		}
		if len(newPass1) < 8 {
			return fmt.Errorf("passphrase must be at least 8 characters")
		}

		newPass2, err := readPassphrase("Confirm new passphrase: ")
		if err != nil {
			return err
		}

		if string(newPass1) != string(newPass2) {
			return fmt.Errorf("passphrases do not match")
		}

		fmt.Print("Changing passphrase...")
		if err := db.ChangePassphrase(engName, oldPass, newPass1); err != nil {
			return err
		}

		crypto.ZeroBytes(oldPass)
		crypto.ZeroBytes(newPass1)
		crypto.ZeroBytes(newPass2)

		fmt.Println(" done.")
		fmt.Println("Passphrase changed successfully.")
		return nil
	},
}
