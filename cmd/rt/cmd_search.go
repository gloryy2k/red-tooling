package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/search"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search evidence by text (input/output)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		query := strings.Join(args, " ")
		tag, _ := cmd.Flags().GetString("tag")
		priority, _ := cmd.Flags().GetString("priority")
		limit, _ := cmd.Flags().GetInt("limit")

		results, err := search.Search(database, engID, query, tag, priority, limit)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Printf("  No results for %q\n", query)
			return nil
		}

		fmt.Printf("  Found %d results for %q:\n\n", len(results), query)
		for _, r := range results {
			tags := ""
			if len(r.Tags) > 0 {
				tags = " [" + strings.Join(r.Tags, ",") + "]"
			}
			pri := ""
			if r.Priority != "" {
				pri = " (" + r.Priority + ")"
			}
			fmt.Printf("  #%-4d %s %s%s%s\n", r.ID, r.Timestamp[:19], r.Action, tags, pri)

			output := r.Output
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			if output != "" {
				for _, line := range strings.Split(output, "\n") {
					fmt.Printf("         %s\n", line)
				}
			}
			fmt.Println()
		}
		return nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query <sql>",
	Short: "Execute a read-only SQL query",
	Long:  "Execute a read-only SQL query against the engagement database. Only SELECT statements allowed.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		q := strings.Join(args, " ")
		results, err := search.Query(database, q)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("  No rows returned")
			return nil
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		}

		// Table output
		if len(results) > 0 {
			cols := make([]string, 0)
			for k := range results[0] {
				cols = append(cols, k)
			}

			// Header
			fmt.Print("  ")
			for _, c := range cols {
				fmt.Printf("%-20s", c)
			}
			fmt.Println()
			fmt.Print("  ")
			for range cols {
				fmt.Print(strings.Repeat("-", 20))
			}
			fmt.Println()

			for _, row := range results {
				fmt.Print("  ")
				for _, c := range cols {
					val := fmt.Sprintf("%v", row[c])
					if len(val) > 19 {
						val = val[:16] + "..."
					}
					fmt.Printf("%-20s", val)
				}
				fmt.Println()
			}
			fmt.Printf("\n  %d rows\n", len(results))
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().String("tag", "", "Filter by tag")
	searchCmd.Flags().String("priority", "", "Filter by priority")
	searchCmd.Flags().Int("limit", 50, "Max results")
	queryCmd.Flags().Bool("json", false, "Output as JSON")
}
