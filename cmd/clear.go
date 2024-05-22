package cmd

import (
	degit "github.com/qiushiyan/go-degit/pkg"
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all existing download caches. Accept an optional argument to filter by the repository",
	Long:  `Clear all existing download caches. Accept an optional argument to filter by the repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var filter string
		if len(args) > 0 {
			filter = args[0]
		}
		return degit.ClearCache(filter, Verbose)
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
