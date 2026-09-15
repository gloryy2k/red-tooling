package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/attachments"
	"github.com/user/rt/internal/findings"
	"github.com/user/rt/internal/remote"
)

var findingCmd = &cobra.Command{
	Use:   "finding <title>",
	Short: "Create a new finding",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		desc, _ := cmd.Flags().GetString("desc")
		priority, _ := cmd.Flags().GetString("priority")
		evFlag, _ := cmd.Flags().GetString("evidence")
		mitreFlag, _ := cmd.Flags().GetString("mitre")
		host, _ := cmd.Flags().GetString("host")
		operator := getOperator()

		var evIDs []int64
		if evFlag != "" {
			for _, s := range strings.Split(evFlag, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid evidence ID: %s", s)
				}
				evIDs = append(evIDs, id)
			}
		}

		var mitre []string
		if mitreFlag != "" {
			for _, m := range strings.Split(mitreFlag, ",") {
				mitre = append(mitre, strings.TrimSpace(m))
			}
		}

		if priority == "" {
			priority = "medium"
		}

		if state, err := remote.LoadState(); err == nil && state != nil {
			return createRemoteFinding(state, title, desc, priority, host, mitre, evIDs, operator)
		}

		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := resolveEngID(database, engName)

		f, err := findings.Create(database, engID, title, desc, priority, operator, host, evIDs, mitre)
		if err != nil {
			return err
		}

		noteFlag, _ := cmd.Flags().GetString("note")
		if noteFlag != "" {
			if err := findings.SetNotes(database, f.ID, noteFlag, operator); err != nil {
				return err
			}
			fmt.Printf("  Finding #%d created: %s [%s] (with PoC notes)\n", f.ID, f.Title, f.Priority)
		} else {
			fmt.Printf("  Finding #%d created: %s [%s]\n", f.ID, f.Title, f.Priority)
			fmt.Printf("  HINT: Add PoC notes with: rt finding-note %d \"Step 1: ... Result: ...\"\n", f.ID)
		}
		return nil
	},
}

var findingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "List findings",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			list, cerr := client.ListFindings()
			if cerr != nil {
				return fmt.Errorf("list findings (remote): %w", cerr)
			}
			if len(list) == 0 {
				fmt.Println("  No findings.")
				return nil
			}
			fmt.Println()
			fmt.Printf("  %-4s %-10s %-12s %-50s\n", "ID", "PRIORITY", "STATUS", "TITLE")
			fmt.Printf("  %-4s %-10s %-12s %-50s\n",
				strings.Repeat("-", 4), strings.Repeat("-", 10), strings.Repeat("-", 12), strings.Repeat("-", 50))
			for _, f := range list {
				title := fmt.Sprintf("%v", f["title"])
				if len(title) > 48 {
					title = title[:45] + "..."
				}
				pri := fmt.Sprintf("%v", f["priority"])
				status := fmt.Sprintf("%v", f["verified"])
				id := f["id"]
				fmt.Printf("  %-4v %-10s %-12s %-50s\n", id, pri, status, title)
			}
			fmt.Printf("\n  %d finding(s) (remote)\n\n", len(list))
			return nil
		}
		engID := resolveEngID(database, engName)

		list, err := findings.List(database, engID)
		if err != nil {
			return err
		}

		if len(list) == 0 {
			fmt.Println("  No findings.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-4s %-10s %-12s %-50s %s\n", "ID", "PRIORITY", "STATUS", "TITLE", "EVIDENCE")
		fmt.Printf("  %-4s %-10s %-12s %-50s %s\n",
			strings.Repeat("-", 4), strings.Repeat("-", 10), strings.Repeat("-", 12),
			strings.Repeat("-", 50), strings.Repeat("-", 10))

		for _, f := range list {
			evStr := fmt.Sprintf("%d", len(f.EvidenceIDs))
			title := f.Title
			if len(title) > 48 {
				title = title[:45] + "..."
			}

			statusColor := ""
			resetColor := "\033[0m"
			switch f.Verified {
			case "confirmed":
				statusColor = "\033[32m"
			case "false-positive":
				statusColor = "\033[90m"
			default:
				statusColor = "\033[33m"
			}

			priColor := priorityColor(f.Priority)
			fmt.Printf("  %-4d %s%-10s\033[0m %s%-12s%s %-50s %s\n",
				f.ID, priColor, f.Priority, statusColor, f.Verified, resetColor, title, evStr)
		}

		uv, cf, fp := findings.CountByStatus(database, engID)
		fmt.Printf("\n  %d finding(s): %d unverified, %d confirmed, %d false-positive\n\n", uv+cf+fp, uv, cf, fp)
		return nil
	},
}

var verifyFindingCmd = &cobra.Command{
	Use:   "verify-finding <id> <confirmed|false-positive>",
	Short: "Mark a finding as confirmed or false-positive",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid finding ID: %s", args[0])
		}

		status := args[1]
		if status != "confirmed" && status != "false-positive" {
			return fmt.Errorf("status must be 'confirmed' or 'false-positive'")
		}

		note, _ := cmd.Flags().GetString("note")
		screenshot, _ := cmd.Flags().GetString("screenshot")
		noScreenshot, _ := cmd.Flags().GetBool("no-screenshot")
		operator := getOperator()

		database, _, dbErr := requireDB()
		if dbErr != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.VerifyFinding(id, status, note); err != nil {
				return fmt.Errorf("verify finding (remote): %w", err)
			}
			fmt.Printf("  [remote] Finding #%d marked as %s\n", id, status)
			return nil
		}

		if status == "confirmed" && screenshot == "" && !noScreenshot {
			fmt.Println("  WARNING: No screenshot provided for confirmed finding.")
			fmt.Println("  Use --screenshot <file> for visual evidence, or --no-screenshot if not applicable.")
		}

		if screenshot != "" {
			if _, err := os.Stat(screenshot); os.IsNotExist(err) {
				return fmt.Errorf("screenshot file not found: %s", screenshot)
			}
			f, err := findings.Get(database, id)
			if err != nil {
				return err
			}
			if len(f.EvidenceIDs) > 0 {
				_, err = attachments.Store(database, f.EvidenceIDs[0], screenshot, fmt.Sprintf("Verification screenshot for finding #%d", id), operator)
				if err != nil {
					return fmt.Errorf("attach screenshot: %w", err)
				}
				fmt.Printf("  Screenshot attached to evidence #%d\n", f.EvidenceIDs[0])
			} else {
				fmt.Println("  WARNING: No linked evidence — screenshot not attached. Use 'rt attach' manually.")
			}
		}

		if err := findings.Verify(database, id, status, operator, note); err != nil {
			return err
		}

		fmt.Printf("  Finding #%d marked as %s\n", id, status)
		return nil
	},
}

var mergeFindingsCmd = &cobra.Command{
	Use:   "merge-findings <id1,id2,...> <new-title>",
	Short: "Merge multiple findings into one",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := requireDB()
		if err != nil {
			return err
		}

		var ids []int64
		for _, s := range strings.Split(args[0], ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid finding ID: %s", s)
			}
			ids = append(ids, id)
		}

		newTitle := strings.Join(args[1:], " ")
		operator := getOperator()

		merged, err := findings.Merge(database, ids, newTitle, operator)
		if err != nil {
			return err
		}

		fmt.Printf("  Merged %d findings → Finding #%d: %s [%s]\n", len(ids), merged.ID, merged.Title, merged.Priority)
		return nil
	},
}

var findingNoteCmd = &cobra.Command{
	Use:   "finding-note <id> <notes>",
	Short: "Set PoC notes for a finding (used in report Proof of Concept section)",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid finding ID: %s", args[0])
		}
		notes := strings.Join(args[1:], " ")

		database, _, dbErr := requireDB()
		if dbErr != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.SetFindingNotes(id, notes); err != nil {
				return fmt.Errorf("set notes (remote): %w", err)
			}
			fmt.Printf("  [remote] PoC notes set for Finding #%d\n", id)
			return nil
		}

		operator := getOperator()
		if err := findings.SetNotes(database, id, notes, operator); err != nil {
			return err
		}

		fmt.Printf("  PoC notes set for Finding #%d\n", id)
		return nil
	},
}

var recommendCmd = &cobra.Command{
	Use:   "recommend <finding-id> <recommendation>",
	Short: "Set a recommendation for a finding",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid finding ID: %s", args[0])
		}
		rec := strings.Join(args[1:], " ")

		database, _, dbErr := requireDB()
		if dbErr != nil {
			if !remote.IsJoined() {
				return fmt.Errorf("no active engagement and not joined to a server — use 'rt join' first")
			}
			client, _, cerr := remoteClientFromState()
			if cerr != nil {
				return cerr
			}
			if err := client.SetRecommendation(id, rec); err != nil {
				return fmt.Errorf("set recommendation (remote): %w", err)
			}
			fmt.Printf("  [remote] Recommendation set for Finding #%d\n", id)
			return nil
		}

		operator := getOperator()
		if err := findings.SetRecommendation(database, id, rec, operator); err != nil {
			return err
		}

		fmt.Printf("  Recommendation set for Finding #%d\n", id)
		return nil
	},
}

func createRemoteFinding(state *remote.ConnectionState, title, desc, priority, host string, mitre []string, evIDs []int64, operator string) error {
	client := remote.NewClient(state.ServerURL, state.APIKey)
	if state.Insecure {
		client.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	resp, err := client.SubmitFinding(remote.FindingReq{
		Title:       title,
		Description: desc,
		Priority:    priority,
		Host:        host,
		Mitre:       mitre,
		EvidenceIDs: evIDs,
		Operator:    operator,
	})
	if err != nil {
		return fmt.Errorf("remote finding: %w", err)
	}
	fmt.Printf("  [remote] Finding #%d created: %s [%s]\n", resp.ID, title, priority)
	return nil
}

func init() {
	findingCmd.Flags().String("desc", "", "Finding description")
	findingCmd.Flags().String("priority", "medium", "Priority: critical, high, medium, low, info")
	findingCmd.Flags().String("evidence", "", "Comma-separated evidence IDs")
	findingCmd.Flags().String("mitre", "", "Comma-separated MITRE ATT&CK IDs")
	findingCmd.Flags().String("note", "", "PoC notes (appears in report Proof of Concept section)")
	findingCmd.Flags().String("host", "", "Target host IP/hostname for this finding")

	verifyFindingCmd.Flags().String("note", "", "Verification note")
	verifyFindingCmd.Flags().String("screenshot", "", "Screenshot file to attach as verification evidence")
	verifyFindingCmd.Flags().Bool("no-screenshot", false, "Skip screenshot requirement for non-visual findings")
}

func priorityColor(p string) string {
	switch p {
	case "critical":
		return "\033[31m"
	case "high":
		return "\033[91m"
	case "medium":
		return "\033[33m"
	case "low":
		return "\033[36m"
	default:
		return "\033[37m"
	}
}
