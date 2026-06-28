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

		if err := destinationConflict(dst, flat, Force, repo.IsFile); err != nil {
			return err
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
				"Output directory is empty: the repository (or subdirectory) contained no files",
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
// repository name (folder mode, no subdir). In --flat folder mode the named
// base is never used: contents land directly in dst (or the cwd when dst is
// omitted).
func resolveDestination(repo *degit.Repo, args []string, flat bool) string {
	if len(args) >= 2 {
		dst := args[1]
		if !flat {
			if stat, err := os.Stat(dst); err == nil && stat.IsDir() {
				return filepath.Join(dst, destinationBase(repo))
			}
		}
		return dst
	}

	if flat {
		return "."
	}
	return destinationBase(repo)
}

// destinationConflict reports whether an already-existing dst blocks the clone.
// It runs before any download so the CLI can fail fast with a clear message:
//
//   - non-flat, no --force: refuse (dst already exists)
//   - non-flat, --force: allowed, except a file target onto an existing dir
//   - flat, no --force: allowed only if dst is a directory (we overlay into it)
//   - flat, --force: allowed (--force wins; Repo.Clone wipes then extracts)
//
// A missing dst (or an unreadable one) is not a conflict here; Repo.Clone makes
// the final decision.
func destinationConflict(dst string, flat, force, isFile bool) error {
	stat, err := os.Stat(dst)
	if err != nil {
		return nil
	}
	switch {
	case flat && !force:
		if !stat.IsDir() {
			return fmt.Errorf("destination `%s` already exists and is not a directory", dst)
		}
	case force:
		if isFile && stat.IsDir() {
			return fmt.Errorf("destination `%s` is a directory; refusing to overwrite with a file", dst)
		}
	default:
		return fmt.Errorf("destination `%s` already exists, use --force to overwrite", dst)
	}
	return nil
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
