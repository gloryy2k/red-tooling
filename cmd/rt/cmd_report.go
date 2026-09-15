package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/remote"
	"github.com/user/rt/internal/report"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate engagement report",
	RunE: func(cmd *cobra.Command, args []string) error {
		htmlFlag, _ := cmd.Flags().GetBool("html")
		execFlag, _ := cmd.Flags().GetBool("exec")
		techFlag, _ := cmd.Flags().GetBool("tech")
		critFlag, _ := cmd.Flags().GetBool("critical")
		verifiedFlag, _ := cmd.Flags().GetBool("verified")
		outFile, _ := cmd.Flags().GetString("output")
		tmplName, _ := cmd.Flags().GetString("template")

		database, engName, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			format := "markdown"
			if htmlFlag {
				format = "html"
			}
			opts := map[string]string{}
			if execFlag {
				opts["exec"] = "true"
			}
			if techFlag {
				opts["tech"] = "true"
			}
			if critFlag {
				opts["critical"] = "true"
			}
			if verifiedFlag {
				opts["verified"] = "true"
			}
			content, cerr := client.GetReport(format, opts)
			if cerr != nil {
				return fmt.Errorf("get report (remote): %w", cerr)
			}
			if outFile != "" {
				if err := os.WriteFile(outFile, []byte(content), 0600); err != nil {
					return fmt.Errorf("write report: %w", err)
				}
				fmt.Printf("  [remote] Report written to: %s\n", outFile)
			} else {
				fmt.Print(content)
			}
			return nil
		}
		engID := resolveEngID(database, engName)

		opts := report.Options{
			ExecOnly:     execFlag,
			TechOnly:     techFlag,
			CriticalOnly: critFlag,
			VerifiedOnly: verifiedFlag,
		}

		data, err := report.Gather(database, engID, opts)
		if err != nil {
			return err
		}

		var missingNotes []int64
		for _, f := range data.Findings {
			if f.Notes == "" {
				missingNotes = append(missingNotes, f.ID)
			}
		}
		if len(missingNotes) > 0 {
			fmt.Printf("  WARNING: %d finding(s) have NO PoC notes (report will have empty Proof of Concept sections)\n", len(missingNotes))
			for _, id := range missingNotes {
				fmt.Printf("    → rt finding-note %d \"Step 1: ... Step 2: ... Result: ...\"\n", id)
			}
			fmt.Println()
		}

		var content string
		if htmlFlag {
			tmplCfg, err := report.LoadTemplate(tmplName)
			if err != nil {
				return err
			}
			content, err = report.RenderTemplateHTML(data, opts, tmplCfg)
			if err != nil {
				return fmt.Errorf("render template: %w", err)
			}
		} else {
			content = report.RenderMarkdown(data, opts)
		}

		if outFile != "" {
			if err := os.WriteFile(outFile, []byte(content), 0600); err != nil {
				return fmt.Errorf("write report: %w", err)
			}
			fmt.Printf("  Report written to: %s\n", outFile)
			fmt.Printf("  Findings: %d | Evidence: %d | Chain: %s\n",
				data.Stats.TotalFindings, data.Stats.TotalEvidence, chainLabel(data.ChainIntact))
		} else {
			fmt.Print(content)
		}

		return nil
	},
}

func init() {
	reportCmd.Flags().Bool("html", false, "Generate HTML report")
	reportCmd.Flags().Bool("exec", false, "Executive summary only")
	reportCmd.Flags().Bool("tech", false, "Technical details only")
	reportCmd.Flags().Bool("critical", false, "Critical and high findings only")
	reportCmd.Flags().Bool("verified", false, "Confirmed findings only")
	reportCmd.Flags().StringP("output", "o", "", "Output file path")
	reportCmd.Flags().String("template", "default", "Report template name (from ~/.rt/templates/)")
}

func chainLabel(ok bool) string {
	if ok {
		return "INTACT"
	}
	return "BROKEN"
}
