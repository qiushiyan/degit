package degit

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func untar(file, dst, subdir, prefix string, isFile bool) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Dir mode appends a slash so HasPrefix matches directory boundaries
	// rather than partial path segments (e.g. "/lib" should match "/lib/foo"
	// but not "/library/foo"). File mode compares the entry name exactly.
	if !isFile && subdir != "" && !strings.HasSuffix(subdir, "/") {
		subdir += "/"
	}

	// Folder-mode bookkeeping: whether any entry fell inside subdir, and the
	// set of directories seen, used to suggest a near-match when subdir is not found.
	matched := false
	var dirs []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Name == "pax_global_header" {
			continue
		}

		header.Name = strings.TrimPrefix(header.Name, prefix)

		if isFile {
			if header.Name != subdir {
				continue
			}
			if header.Typeflag == tar.TypeDir {
				return fmt.Errorf("path %s is a directory, not a file", strings.TrimPrefix(subdir, "/"))
			}
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return err
			}
			out, err := os.Create(dst)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := os.Chmod(dst, os.FileMode(header.Mode)); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}

		// Only collected to build a suggestion if subdir turns out to be
		// missing; skip once we've matched, and entirely for whole-repo clones.
		if subdir != "" && !matched && header.Typeflag == tar.TypeDir {
			dirs = append(dirs, header.Name)
		}

		if subdir != "" {
			if !strings.HasPrefix(header.Name, subdir) {
				continue
			}
			header.Name = strings.TrimPrefix(header.Name, subdir)
		}
		matched = true

		target := filepath.Join(dst, header.Name)

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}

	if isFile {
		return fmt.Errorf("file not found in repository: %s", strings.TrimPrefix(subdir, "/"))
	}

	if subdir != "" && !matched {
		return notFoundDirError(subdir, dirs)
	}

	return nil
}

// notFoundDirError builds a helpful error when a requested subdir matched no
// entries. It suggests archive directories whose path ends with the requested
// subdir, or failing that whose final segment matches it — which catches the
// common case of omitting an intermediate folder (asking for
// "productivity/grill-me" when the real path is "skills/productivity/grill-me").
func notFoundDirError(subdir string, dirs []string) error {
	want := strings.Trim(subdir, "/")
	wantBase := path.Base(want)

	var suffix, base []string
	seen := map[string]bool{}
	for _, d := range dirs {
		clean := strings.Trim(d, "/")
		if clean == "" || clean == want || seen[clean] {
			continue
		}
		seen[clean] = true
		switch {
		case strings.HasSuffix(clean, "/"+want):
			suffix = append(suffix, clean)
		case path.Base(clean) == wantBase:
			base = append(base, clean)
		}
	}

	suggestions := append(suffix, base...)
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	if len(suggestions) > 0 {
		return fmt.Errorf(
			"directory not found in repository: %s (did you mean: %s?)",
			want, strings.Join(suggestions, ", "),
		)
	}
	return fmt.Errorf("directory not found in repository: %s", want)
}
