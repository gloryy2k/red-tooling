package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/search"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show database table and column names",
	Long:  "List all tables and their columns in the engagement database. Useful for crafting rt query SQL statements.",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		tables, err := search.Query(database, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
		if err != nil {
			return err
		}

		if len(tables) == 0 {
			fmt.Println("  No tables found")
			return nil
		}

		fmt.Println()
		for _, t := range tables {
			tableName := fmt.Sprintf("%v", t["name"])
			if strings.HasPrefix(tableName, "sqlite_") {
				continue
			}

			rows, err := database.Query(fmt.Sprintf("PRAGMA table_info('%s')", tableName))
			if err != nil {
				fmt.Printf("  %s — (error reading columns)\n", tableName)
				continue
			}

			var colNames []string
			for rows.Next() {
				var cid int
				var name, typ string
				var notNull int
				var dflt interface{}
				var pk int
				if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
					continue
				}
				entry := name
				if typ != "" {
					entry += " " + typ
				}
				colNames = append(colNames, entry)
			}
			rows.Close()

			fmt.Printf("  %s\n", tableName)
			for _, cn := range colNames {
				fmt.Printf("    %s\n", cn)
			}
			fmt.Println()
		}
		return nil
	},
}
