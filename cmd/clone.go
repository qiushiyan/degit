package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

		// --flat only applies to folder downloads; a single file is always
		// written to its resolved path regardless.
		flat := Flat && !repo.IsFile
		repo.Flat = flat

		dst := resolveDestination(repo, args, flat)

		// In flat mode we deliberately overlay onto an existing dir, so the
		// dst-exists guard is skipped.
		if stat, err := os.Stat(dst); err == nil && !flat {
			if !Force {
				return fmt.Errorf("destination `%s` already exists, use --force to overwrite", dst)
			}
			if repo.IsFile && stat.IsDir() {
				return fmt.Errorf("destination `%s` is a directory; refusing to overwrite with a file", dst)
			}
		}

		if Verbose {
			fmt.Fprintf(os.Stderr, "Cloning `%s` into `%s`\n", repo.URL, dst)
		}

		if err := repo.Resolve(); err != nil {
			return err
		}

		if !Quiet {
			if repo.Cached {
				printCacheHit(os.Stderr, repo)
			} else {
				printResolved(os.Stderr, repo)
			}
		}

		if !Quiet && !NoProgress && !repo.Cached && term.IsTerminal(int(os.Stderr.Fd())) {
			repo.Progress = newCLIProgress("downloading")
		}

		if err := repo.Clone(dst, Force, Verbose); err != nil {
			return err
		}

		if !Quiet {
			printDone(os.Stderr, repo, dst)
		}

		if repo.IsFile {
			return nil
		}

		entries, err := os.ReadDir(dst)
		if err != nil {
			return err
		}
		if len(entries) == 0 && !Quiet {
			fmt.Fprintln(
				os.Stderr,
				"Output directory is empty, you might have specified an non-existing subfolder in the repository",
			)
		}
		return nil
	},
}

// resolveDestination applies cp-like semantics, uniformly for files and
// folders:
//
//   - dst omitted             -> base name in the current directory
//   - dst is an existing dir  -> base name created inside it
//   - dst is anything else    -> dst is the literal target (rename)
//
// base is the file's name (file mode), the last segment of the subdir, or the
// repository name (folder mode, no subdir). In --flat folder mode the
// existing-dir rule is skipped so contents land directly in dst.
func resolveDestination(repo *degit.Repo, args []string, flat bool) string {
	base := destinationBase(repo)

	if len(args) >= 2 {
		dst := args[1]
		if !flat {
			if stat, err := os.Stat(dst); err == nil && stat.IsDir() {
				return filepath.Join(dst, base)
			}
		}
		return dst
	}

	return base
}

// destinationBase is the name the download lands under when dst is omitted or
// is an existing directory: the trailing path segment of a file/subdir target,
// or the repository name when cloning a whole repo.
func destinationBase(repo *degit.Repo) string {
	if repo.Subdir != "" {
		return filepath.Base(strings.TrimPrefix(repo.Subdir, "/"))
	}
	return repo.Name
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
