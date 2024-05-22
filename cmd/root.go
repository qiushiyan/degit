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
	Long: `A Go port of the node degit cli.

	degit user/repo#ref output-dir

downloads the github repository locally. You can specify subdirectories and use Gitlab and Bitbucket repositories as well. If the commit hash does not change, degit uses the cached version to save downloading again.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
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
