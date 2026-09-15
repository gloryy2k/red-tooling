package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/agentctx"
	"github.com/user/rt/internal/remote"
)

const rtBeginMarker = "<!-- RT:BEGIN -->"
const rtEndMarker = "<!-- RT:END -->"

var initWorkspaceCmd = &cobra.Command{
	Use:   "init-workspace",
	Short: "Add RT agent context to current directory (appends to CLAUDE.md + creates skills)",
	Long: `Appends RT agent context to CLAUDE.md (between RT:BEGIN/RT:END markers) and
creates .claude/commands/ skills. If CLAUDE.md already has RT markers, they are
replaced. If no CLAUDE.md exists, one is created. Existing content outside
the markers is preserved.

Run this in your pentest working directory before starting a session.
Use 'rt remove-workspace' to strip RT content from the workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()

		content := agentctx.GenerateClaudeMDContent("", "", "")

		if remote.IsJoined() {
			state, err := remote.LoadState()
			if err == nil {
				content += fmt.Sprintf("\n## Active Server Connection\n\n")
				content += fmt.Sprintf("**Server:** `%s`\n", state.ServerURL)
				content += fmt.Sprintf("**Operator:** `%s`\n", state.Operator)
				content += fmt.Sprintf("**Session:** `%s`\n\n", state.SessionID)
				content += "You are already joined. Just run `rt exec` — evidence auto-syncs.\n"
				fmt.Println("  Detected remote mode (joined server)")
			}
		}

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if err := appendOrReplaceRT(claudeMD, content); err != nil {
			return fmt.Errorf("update CLAUDE.md: %w", err)
		}
		fmt.Printf("  Updated: %s (RT section appended)\n", claudeMD)

		skillsDir := filepath.Join(dir, ".claude", "commands")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("create skills dir: %w", err)
		}

		skills := agentctx.SkillFiles()
		for name, content := range skills {
			p := filepath.Join(skillsDir, name)
			if err := os.WriteFile(p, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
			fmt.Printf("  Created: %s\n", p)
		}

		fmt.Println()
		fmt.Println("  Workspace ready. Open Claude Code and give it a target.")
		fmt.Println("  Skills: /recon, /exploit, /post-exploit, /report, /standup")
		fmt.Println()
		fmt.Println("  Use 'rt remove-workspace' to strip RT content.")

		return nil
	},
}

var removeWorkspaceCmd = &cobra.Command{
	Use:   "remove-workspace",
	Short: "Remove RT agent context from current directory",
	Long:  `Strips the RT:BEGIN..RT:END section from CLAUDE.md and removes RT skill files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()

		claudeMD := filepath.Join(dir, "CLAUDE.md")
		if err := stripRTSection(claudeMD); err != nil {
			fmt.Printf("  CLAUDE.md: %s\n", err)
		} else {
			fmt.Printf("  Cleaned: %s (RT section removed)\n", claudeMD)
		}

		skillsDir := filepath.Join(dir, ".claude", "commands")
		for name := range agentctx.SkillFiles() {
			p := filepath.Join(skillsDir, name)
			if err := os.Remove(p); err == nil {
				fmt.Printf("  Removed: %s\n", p)
			}
		}

		fmt.Println("  RT workspace content removed.")
		return nil
	},
}

func appendOrReplaceRT(path, rtContent string) error {
	block := "\n" + rtBeginMarker + "\n" + rtContent + "\n" + rtEndMarker + "\n"

	existing, err := os.ReadFile(path)
	if err != nil {
		return os.WriteFile(path, []byte(strings.TrimLeft(block, "\n")), 0644)
	}

	content := string(existing)
	beginIdx := strings.Index(content, rtBeginMarker)
	endIdx := strings.Index(content, rtEndMarker)

	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		content = content[:beginIdx] + strings.TrimLeft(block, "\n") + content[endIdx+len(rtEndMarker):]
		content = strings.TrimRight(content, "\n") + "\n"
		return os.WriteFile(path, []byte(content), 0644)
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	return os.WriteFile(path, []byte(content), 0644)
}

func stripRTSection(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("not found")
	}
	content := string(data)
	beginIdx := strings.Index(content, rtBeginMarker)
	endIdx := strings.Index(content, rtEndMarker)
	if beginIdx < 0 || endIdx < 0 {
		return fmt.Errorf("no RT section found")
	}

	before := strings.TrimRight(content[:beginIdx], "\n")
	after := strings.TrimLeft(content[endIdx+len(rtEndMarker):], "\n")
	result := before
	if after != "" {
		result += "\n\n" + after
	}
	result = strings.TrimSpace(result) + "\n"
	if strings.TrimSpace(result) == "" {
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(result), 0644)
}
