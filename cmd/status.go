package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
)

// printResolved announces a download that's about to start. In file mode it
// names the target file rather than showing the short hash, because the
// user usually cares more about which file is being saved than which commit.
func printResolved(w io.Writer, r *degit.Repo) {
	if r.IsFile {
		name := filepath.Base(strings.TrimPrefix(r.Subdir, "/"))
		fmt.Fprintf(w, "↓ %s/%s@%s → %s\n", r.User, r.Name, r.Ref, name)
		return
	}
	fmt.Fprintf(w, "↓ %s/%s@%s (%s)\n", r.User, r.Name, r.Ref, shortHash(r.Hash))
}

// printCacheHit announces that the tarball was already on disk.
func printCacheHit(w io.Writer, r *degit.Repo) {
	fmt.Fprintf(w, "↪ using cache %s/%s@%s (%s)\n",
		r.User, r.Name, r.Ref, shortHash(r.Hash))
}

// printDone confirms successful extraction.
func printDone(w io.Writer, r *degit.Repo, dst string) {
	if r.IsFile {
		fmt.Fprintf(w, "✓ saved to %s\n", dst)
		return
	}
	fmt.Fprintf(w, "✓ extracted to %s\n", dst)
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
