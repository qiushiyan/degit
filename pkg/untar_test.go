package degit

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name    string
	content string
	isDir   bool
	mode    int64
}

func writeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode}
		if e.isDir {
			hdr.Typeflag = tar.TypeDir
			if hdr.Mode == 0 {
				hdr.Mode = 0o755
			}
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(e.content))
			if hdr.Mode == 0 {
				hdr.Mode = 0o644
			}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader %q: %v", e.name, err)
		}
		if !e.isDir {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("Write %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(b)
}

func TestUntarFullExtraction(t *testing.T) {
	prefix := "proj-abc"
	src := writeTarGz(t, []tarEntry{
		{name: "proj-abc/", isDir: true},
		{name: "proj-abc/README.md", content: "hello"},
		{name: "proj-abc/lib/", isDir: true},
		{name: "proj-abc/lib/foo.go", content: "package foo"},
	})
	dst := t.TempDir()

	if err := untar(src, dst, "", prefix, false); err != nil {
		t.Fatalf("untar: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "README.md")); got != "hello" {
		t.Errorf("README.md content: got %q, want %q", got, "hello")
	}
	if got := readFile(t, filepath.Join(dst, "lib/foo.go")); got != "package foo" {
		t.Errorf("lib/foo.go content: got %q, want %q", got, "package foo")
	}
}

func TestUntarSubdir(t *testing.T) {
	prefix := "proj-abc"
	src := writeTarGz(t, []tarEntry{
		{name: "proj-abc/", isDir: true},
		{name: "proj-abc/README.md", content: "hello"},
		{name: "proj-abc/lib/", isDir: true},
		{name: "proj-abc/lib/foo.go", content: "package foo"},
		{name: "proj-abc/lib/sub/", isDir: true},
		{name: "proj-abc/lib/sub/bar.go", content: "package sub"},
	})
	dst := t.TempDir()

	if err := untar(src, dst, "/lib", prefix, false); err != nil {
		t.Fatalf("untar: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "foo.go")); got != "package foo" {
		t.Errorf("foo.go content: got %q, want %q", got, "package foo")
	}
	if got := readFile(t, filepath.Join(dst, "sub/bar.go")); got != "package sub" {
		t.Errorf("sub/bar.go content: got %q, want %q", got, "package sub")
	}
	if _, err := os.Stat(filepath.Join(dst, "README.md")); !os.IsNotExist(err) {
		t.Errorf("README.md should not be extracted under subdir=/lib")
	}
}

func TestUntarSingleFile(t *testing.T) {
	prefix := "proj-abc"
	src := writeTarGz(t, []tarEntry{
		{name: "proj-abc/", isDir: true},
		{name: "proj-abc/README.md", content: "hello", mode: 0o640},
		{name: "proj-abc/other.txt", content: "other"},
		{name: "proj-abc/lib/", isDir: true},
		{name: "proj-abc/lib/foo.go", content: "package foo"},
	})
	outDir := t.TempDir()
	dst := filepath.Join(outDir, "out.md")

	if err := untar(src, dst, "/README.md", prefix, true); err != nil {
		t.Fatalf("untar: %v", err)
	}

	if got := readFile(t, dst); got != "hello" {
		t.Errorf("dst content: got %q, want %q", got, "hello")
	}
	for _, leak := range []string{"other.txt", "lib", "lib/foo.go"} {
		if _, err := os.Stat(filepath.Join(outDir, leak)); !os.IsNotExist(err) {
			t.Errorf("unexpected entry leaked into outDir in file mode: %s", leak)
		}
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("mode: got %o, want %o", got, 0o640)
	}
}

func TestUntarSingleFileNested(t *testing.T) {
	prefix := "proj-abc"
	src := writeTarGz(t, []tarEntry{
		{name: "proj-abc/", isDir: true},
		{name: "proj-abc/lib/", isDir: true},
		{name: "proj-abc/lib/foo.go", content: "package foo"},
	})
	dst := filepath.Join(t.TempDir(), "dir-that-does-not-exist-yet", "out.go")

	if err := untar(src, dst, "/lib/foo.go", prefix, true); err != nil {
		t.Fatalf("untar: %v", err)
	}
	if got := readFile(t, dst); got != "package foo" {
		t.Errorf("content: got %q, want %q", got, "package foo")
	}
}

func TestUntarSingleFileNotFound(t *testing.T) {
	prefix := "proj-abc"
	src := writeTarGz(t, []tarEntry{
		{name: "proj-abc/", isDir: true},
		{name: "proj-abc/README.md", content: "hello"},
	})
	dst := filepath.Join(t.TempDir(), "out.md")

	err := untar(src, dst, "/does-not-exist.md", prefix, true)
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
	if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
		t.Errorf("dst should not exist when file is not found")
	}
}
