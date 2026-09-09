package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/agent"
)

var agentRunCmd = &cobra.Command{
	Use:   "agent-run <playbook>",
	Short: "Run an agent playbook",
	Long:  "Execute a YAML playbook from ~/.rt/playbooks/ or a file path.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		playbookName := args[0]
		pb, err := agent.LoadPlaybook(playbookName)
		if err != nil {
			return err
		}

		// Parse --var key=value flags
		vars := make(map[string]string)
		varFlags, _ := cmd.Flags().GetStringSlice("var")
		for _, v := range varFlags {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}

		// Also parse --target shorthand
		target, _ := cmd.Flags().GetString("target")
		if target != "" {
			vars["target"] = target
		}
		domain, _ := cmd.Flags().GetString("domain")
		if domain != "" {
			vars["domain"] = domain
		}

		result, err := agent.RunPlaybook(database, engID, pb, vars)
		if err != nil {
			return err
		}

		fmt.Printf("  Summary: %d/%d steps completed, %d evidence entries\n",
			result.StepsRun, result.StepsTotal, len(result.EvidenceIDs))
		return nil
	},
}

var agentExecCmd = &cobra.Command{
	Use:   "agent-exec <command>",
	Short: "Run a single command as an agent",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		command := strings.Join(args, " ")

		_, err = agent.RunSingleCommand(database, engID, command)
		return err
	},
}

var agentScriptCmd = &cobra.Command{
	Use:   "agent-script <script-path>",
	Short: "Run a script file as an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		_, err = agent.RunScript(database, engID, args[0])
		return err
	},
}

var agentListCmd = &cobra.Command{
	Use:   "agent-list",
	Short: "List available playbooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		playbooks, err := agent.ListPlaybooks()
		if err != nil {
			return err
		}

		if len(playbooks) == 0 {
			fmt.Println("  No playbooks found in ~/.rt/playbooks/")
			fmt.Println("  Create a YAML playbook or run 'rt agent-run <path-to-file>'")
			return nil
		}

		fmt.Println()
		for _, name := range playbooks {
			pb, err := agent.LoadPlaybook(name)
			if err != nil {
				fmt.Printf("  %-20s (error loading)\n", name)
				continue
			}
			desc := pb.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Printf("  %-20s %s (%d steps)\n", name, desc, len(pb.Steps))
		}
		fmt.Println()
		return nil
	},
}

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show agent token usage and cost",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}
		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))

		summary, err := agent.GetSummary(database, engID)
		if err != nil {
			return err
		}

		if summary.EntryCount == 0 {
			fmt.Println("  No agent cost data recorded.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  Agent Cost Summary — %s\n\n", engName)
		fmt.Printf("  %-25s %-12s %-12s %-10s %s\n", "MODEL", "INPUT", "OUTPUT", "CALLS", "COST")
		fmt.Printf("  %-25s %-12s %-12s %-10s %s\n",
			strings.Repeat("-", 25), strings.Repeat("-", 12), strings.Repeat("-", 12),
			strings.Repeat("-", 10), strings.Repeat("-", 10))

		for _, mc := range summary.ByModel {
			fmt.Printf("  %-25s %-12s %-12s %-10d %s\n",
				mc.Model, agent.FormatTokens(mc.InputTokens), agent.FormatTokens(mc.OutputTokens),
				mc.Calls, agent.FormatCost(mc.CostUSD))
		}

		fmt.Printf("\n  Total: %s input, %s output, %d calls → %s\n\n",
			agent.FormatTokens(summary.TotalInput), agent.FormatTokens(summary.TotalOutput),
			summary.EntryCount, agent.FormatCost(summary.TotalCostUSD))

		return nil
	},
}

func init() {
	agentRunCmd.Flags().StringSlice("var", nil, "Variable in key=value format (repeatable)")
	agentRunCmd.Flags().String("target", "", "Target (shorthand for --var target=...)")
	agentRunCmd.Flags().String("domain", "", "Domain (shorthand for --var domain=...)")
}
