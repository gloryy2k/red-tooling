package main

import (
	"database/sql"
	"fmt"

	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/crypto"
	"github.com/user/rt/internal/db"
)

// requireDB returns an open database for the active engagement.
// If the DB is already unlocked (plain file exists), opens it directly.
// Otherwise prompts for passphrase.
func requireDB() (*sql.DB, string, error) {
	engName := config.GetActiveEngagement()
	if engName == "" {
		return nil, "", fmt.Errorf("no active engagement. Run 'rt new' or 'rt use' first")
	}

	// Already open in this process
	if database := db.Current(); database != nil {
		return database, engName, nil
	}

	// Try opening unlocked db (plain file exists from prior rt unlock)
	if db.IsUnlocked(engName) {
		database, err := db.OpenPlain(engName)
		if err != nil {
			return nil, "", err
		}
		return database, engName, nil
	}

	// Need passphrase
	pass, err := readPassphrase("Enter passphrase: ")
	if err != nil {
		return nil, "", fmt.Errorf("read passphrase: %w", err)
	}

	database, err := db.Unlock(engName, pass)
	if err != nil {
		return nil, "", err
	}
	crypto.ZeroBytes(pass)
	return database, engName, nil
}
