package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfdr",
	Short: "tfdr is a tool for performing Terraform state file surgery",
	Long: `A tool that automates state file tedium after
performing module refactors.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(cmd.UsageString())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO:
	// Moving a state file after changing a directory (and moving the lock file)
	// Deleting a statefile and a lock file after destroying the module
	// or when you want to start fresh
}
