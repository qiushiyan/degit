package cmd

import (
	"fmt"
	"os"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone <src> <dst>",
	Short: "Clone a repository locally",
	Long:  `Downloads a repository into a local destination directory.`,
	Args:  cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		dst := args[1]
		if !Force {
			stat, err := os.Stat(dst)
			if err == nil && stat.IsDir() {
				return fmt.Errorf("destination %s already exists, use --force to overwrite", dst)
			}
		}

		err := degit.Clone(args[0], args[1], Force, Verbose)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(dst)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("output directory is empty. did you specify the correct directory?")
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
