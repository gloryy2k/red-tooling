package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/export"
)

var exportCmd = &cobra.Command{
	Use:   "export <format>",
	Short: "Export findings (ghostwriter, csv, json, mitre)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		operator := getOperator()
		outFile, _ := cmd.Flags().GetString("output")

		format := strings.ToLower(args[0])
		var content string

		switch format {
		case "ghostwriter":
			content, err = export.ExportGhostwriter(database, engID, operator)
		case "csv":
			content, err = export.ExportCSV(database, engID, operator)
		case "json":
			content, err = export.ExportJSON(database, engID, operator)
		case "mitre":
			content, err = export.ExportMITRE(database, engID, operator)
		default:
			return fmt.Errorf("unknown format: %s. Supported: ghostwriter, csv, json, mitre", format)
		}

		if err != nil {
			return err
		}

		if outFile != "" {
			if err := os.WriteFile(outFile, []byte(content), 0600); err != nil {
				return fmt.Errorf("write export: %w", err)
			}
			fmt.Printf("  Exported %s → %s\n", format, outFile)
		} else {
			fmt.Print(content)
		}

		return nil
	},
}

func init() {
	exportCmd.Flags().StringP("output", "o", "", "Output file path")
}
