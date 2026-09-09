package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/importer"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import evidence from external tools (nmap XML, nuclei JSON, CSV, text)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		operator := getOperator()
		result, err := importer.Import(database, engID, args[0], operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Imported %d entries from %s (format: %s)\n", result.Entries, result.Source, result.Format)
		return nil
	},
}
