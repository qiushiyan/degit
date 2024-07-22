package cmd

import (
	"fmt"
	"os"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone <src> <dst>",
	Short: "Clone a repository locally",
	Long:  `Downloads a repository into a local destination directory.`,
	Args:  cobra.MatchAll(cobra.OnlyValidArgs, cobra.MinimumNArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := degit.ParseRepo(args[0])
		if err != nil {
			return err
		}

		// if dst is not specified, use the repo name or subdir as the name for the output directory
		var dst string
		if len(args) < 2 {
			if repo.Subdir != "" {
				if strings.HasPrefix(repo.Subdir, "/") {
					dst = repo.Subdir[1:]
				} else {
					dst = repo.Subdir
				}
			} else {
				dst = repo.Name
			}
		} else {
			dst = args[1]
		}

		if !Force {
			stat, err := os.Stat(dst)
			if err == nil && stat.IsDir() {
				return fmt.Errorf("destination `%s` already exists, use --force to overwrite", dst)
			}
		}

		if Verbose {
			fmt.Printf("Cloning `%s` into `%s`\n", repo.URL, dst)
		}

		err = repo.Clone(dst, Force, Verbose)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(dst)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println(
				"Output directory is empty, you might have specified an non-existing subfolder in the repository",
			)
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
