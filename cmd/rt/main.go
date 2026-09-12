package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/db"
)

var rootCmd = &cobra.Command{
	Use:   "rt",
	Short: "RT — Red Team Evidence Logger",
	Long:  `RT captures evidence automatically during red team engagements and generates reports. "Hack more, document less."`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.EnsureDirs()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		db.Close()
	},
}

func main() {
	rootCmd.AddCommand(
		newCmd,
		lsCmd,
		useCmd,
		startCmd,
		stopCmd,
		tagCmd,
		milestoneCmd,
		bookmarkCmd,
		noteCmd,
		timelineCmd,
		unlockCmd,
		lockCmd,
		passwdCmd,
		statusCmd,
		verifyChainCmd,
		execCmd,
		credCmd,
		credsCmd,
		auditCmd,
		findingCmd,
		findingsCmd,
		findingNoteCmd,
		verifyFindingCmd,
		mergeFindingsCmd,
		recommendCmd,
		screenshotCmd,
		attachCmd,
		attachmentsCmd,
		exportAttachCmd,
		deleteCmd,
		redactCmd,
		reportCmd,
		exportCmd,
		serveCmd,
		operatorAddCmd,
		operatorListCmd,
		operatorRotateCmd,
		agentRunCmd,
		agentExecCmd,
		agentScriptCmd,
		agentListCmd,
		costCmd,
		scopeCmd,
		scopeListCmd,
		scopeTestedCmd,
		scopeUntestedCmd,
		checklistCmd,
		checklistLoadCmd,
		checklistAddCmd,
		checkCmd,
		uncheckCmd,
		searchCmd,
		queryCmd,
		importCmd,
		standupCmd,
		timeCmd,
		wipeCmd,
		remoteExecCmd,
		remoteSessionCmd,
		joinCmd,
		leaveCmd,
		syncCmd,
		initWorkspaceCmd,
		removeWorkspaceCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
