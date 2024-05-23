package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command
var clearCmd = &cobra.Command{
	Use:   "clear [filter]",
	Short: "Clear download caches",
	Long:  `Clear all existing download caches. Accept an optional argument to filter by the repository.`,
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1)),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filter string
		if len(args) > 0 {
			filter = args[0]
		}

		dir := degit.GetCacheDir()
		if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
			fmt.Println("no cache found, skipping")
			return nil
		}

		var confirm bool

		if filter == "" {
			count, err := countRepoLevelDirectories(dir)
			if err != nil {
				return err
			}
			err = survey.AskOne(
				&survey.Confirm{
					Message: fmt.Sprintf(
						"Are you sure you want to clear caches for %d repositories?",
						count,
					),
				},
				&confirm,
			)
			if err != nil {
				return err
			}
		} else {
			r, err := degit.ParseRepo(filter)
			if err != nil {
				return err
			}

			dir := path.Join(dir, r.Site, r.User, r.Name)
			if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
				fmt.Printf("no cache found for %s\n", filter)
				return nil
			}

			err = survey.AskOne(
				&survey.Confirm{
					Message: fmt.Sprintf("Are you sure you want to clear cache for %s?", filter),
				},
				&confirm,
			)

			if err != nil {
				return err
			}
		}

		if confirm {
			return degit.ClearCache(filter, Verbose)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}

func countRepoLevelDirectories(root string) (int, error) {
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// site/user/repo
		if info.IsDir() && depth(root, path) == 3 {
			count++
		}
		return nil
	})
	return count, err
}

func depth(root, path string) int {
	return len(strings.Split(strings.TrimPrefix(path, root), "/"))
}
