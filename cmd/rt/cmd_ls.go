package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List engagements",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		entries, err := os.ReadDir(dataDir)
		if err != nil {
			fmt.Println("No engagements found.")
			return nil
		}

		active := config.GetActiveEngagement()
		found := false

		fmt.Println()
		fmt.Printf("  %-3s %-25s %s\n", "", "ENGAGEMENT", "STATUS")
		fmt.Printf("  %-3s %-25s %s\n", "", strings.Repeat("-", 25), strings.Repeat("-", 10))

		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".db.enc") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".db.enc")
			info, _ := e.Info()

			marker := "  "
			if name == active {
				marker = "* "
			}

			size := ""
			if info != nil {
				size = formatSize(info.Size())
			}

			fmt.Printf("  %s%-25s %s\n", marker, name, size)
			found = true
		}

		if !found {
			fmt.Println("  No engagements found.")
			fmt.Println()
			fmt.Println("  Create one with: rt new \"ENGAGEMENT-NAME\"")
		}
		fmt.Println()
		return nil
	},
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

func engagementExists(name string) bool {
	p := filepath.Join(config.DataDir(), name+".db.enc")
	_, err := os.Stat(p)
	return err == nil
}
