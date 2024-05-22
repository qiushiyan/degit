package degit

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func untar(file string, dst string, subdir string, prefix string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a gzip reader
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Create a tar reader
	tr := tar.NewReader(gzr)

	// Normalize subdir to ensure it ends with a slash
	if subdir != "" && !strings.HasSuffix(subdir, "/") {
		subdir += "/"
	}

	// Iterate over the files in the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Skip the pax_global_header file
		if header.Name == "pax_global_header" {
			continue
		}

		header.Name = strings.TrimPrefix(header.Name, prefix)
		if subdir != "" {
			if !strings.HasPrefix(header.Name, subdir) {
				continue
			}
			header.Name = strings.TrimPrefix(header.Name, subdir)
		}
		// Create the output file path
		target := filepath.Join(dst, header.Name)

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Create the parent directories if needed
		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return err
		}

		// Create the output file
		outFile, err := os.Create(target)
		if err != nil {
			return err
		}

		// Copy the file contents to the output file
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return err
		}

		// Set the file permissions
		if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
			outFile.Close()
			return err
		}

		outFile.Close()
	}

	return nil
}
