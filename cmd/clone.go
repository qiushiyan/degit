package cmd

import (
	"fmt"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone <src> <dst>",
	Short: "Clone a repository into a local destination directory",
	Long:  `Clone a repository into a local destination directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("clone requires two arguments")
		}
		repo, err := degit.Parse(args[0])
		if err != nil {
			return err
		}
		err = repo.Clone(args[1], Force, Verbose)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cloneCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cloneCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
