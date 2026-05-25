package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
)

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

		dst := resolveDestination(repo, args)

		if stat, err := os.Stat(dst); err == nil {
			if !Force {
				return fmt.Errorf("destination `%s` already exists, use --force to overwrite", dst)
			}
			if repo.IsFile && stat.IsDir() {
				return fmt.Errorf("destination `%s` is a directory; refusing to overwrite with a file", dst)
			}
		}

		if Verbose {
			fmt.Printf("Cloning `%s` into `%s`\n", repo.URL, dst)
		}

		if err := repo.Clone(dst, Force, Verbose); err != nil {
			return err
		}

		if repo.IsFile {
			return nil
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

// resolveDestination applies cp-like semantics for file targets:
//   - omitted dst: file basename (or subdir / repo name for folder targets)
//   - dst is an existing directory: write inside it using the file's basename
//   - otherwise: dst is the literal target path
func resolveDestination(repo *degit.Repo, args []string) string {
	if len(args) >= 2 {
		dst := args[1]
		if repo.IsFile {
			if stat, err := os.Stat(dst); err == nil && stat.IsDir() {
				return filepath.Join(dst, filepath.Base(strings.TrimPrefix(repo.Subdir, "/")))
			}
		}
		return dst
	}
	if repo.IsFile {
		return filepath.Base(strings.TrimPrefix(repo.Subdir, "/"))
	}
	if repo.Subdir != "" {
		return strings.TrimPrefix(repo.Subdir, "/")
	}
	return repo.Name
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
