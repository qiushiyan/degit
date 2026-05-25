package degit

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
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

		if subdir != "" {
			if !strings.HasPrefix(header.Name, subdir) {
				continue
			}
			header.Name = strings.TrimPrefix(header.Name, subdir)
		}

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

	return nil
}
