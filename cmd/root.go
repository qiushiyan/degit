package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var Verbose bool
var Force bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "degit user/repo#ref output-dir",
	Short: "Straightforward project scaffolding",
	Long: `A Go port of the node degit cli https://github.com/rich-harris/degit.

Usage:

	degit user/repo#ref output-dir

This will download a tarball for the repository github.com/user/repo at "ref" locally, and extracts it to output-dir. You can specify subdirectories and use Gitlab and Bitbucket repositories as well. degit also maintains a cache of downloaded tarballs that can be cleared with "degit clear".`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func Subcommands() (commandNames []string) {
	for _, command := range rootCmd.Commands() {
		commandNames = append(commandNames, append(command.Aliases, command.Name())...)
	}
	return
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().
		BoolVarP(&Force, "force", "f", false, "overwrite destination directory")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceUsage = true
}
