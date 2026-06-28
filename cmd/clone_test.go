package cmd

import (
	"os"
	"path/filepath"
	"testing"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/stretchr/testify/require"
)

func TestResolveDestination(t *testing.T) {
	existing := t.TempDir()                     // an existing directory
	fresh := filepath.Join(existing, "newname") // a path that does not exist yet

	tests := []struct {
		name string
		repo *degit.Repo
		args []string
		flat bool
		want string
	}{
		{
			name: "folder, no subdir, dst omitted -> repo name in cwd",
			repo: &degit.Repo{Name: "repo"},
			args: []string{"src"},
			want: "repo",
		},
		{
			name: "folder, subdir, dst omitted -> subdir basename in cwd",
			repo: &degit.Repo{Name: "repo", Subdir: "/a/grill-me"},
			args: []string{"src"},
			want: "grill-me",
		},
		{
			name: "folder, subdir, dst is existing dir -> named subfolder inside",
			repo: &degit.Repo{Name: "repo", Subdir: "/a/grill-me"},
			args: []string{"src", existing},
			want: filepath.Join(existing, "grill-me"),
		},
		{
			name: "folder, no subdir, dst is existing dir -> repo name inside",
			repo: &degit.Repo{Name: "repo"},
			args: []string{"src", existing},
			want: filepath.Join(existing, "repo"),
		},
		{
			name: "folder, subdir, dst is a fresh path -> literal (rename)",
			repo: &degit.Repo{Name: "repo", Subdir: "/a/grill-me"},
			args: []string{"src", fresh},
			want: fresh,
		},
		{
			name: "folder, subdir, dst is existing dir, --flat -> literal (overlay)",
			repo: &degit.Repo{Name: "repo", Subdir: "/a/grill-me"},
			args: []string{"src", existing},
			flat: true,
			want: existing,
		},
		{
			name: "folder, subdir, dst omitted, --flat -> cwd",
			repo: &degit.Repo{Name: "repo", Subdir: "/a/grill-me"},
			args: []string{"src"},
			flat: true,
			want: ".",
		},
		{
			name: "file, dst omitted -> filename in cwd",
			repo: &degit.Repo{Name: "repo", Subdir: "/docs/README.md", IsFile: true},
			args: []string{"src"},
			want: "README.md",
		},
		{
			name: "file, dst is existing dir -> filename inside",
			repo: &degit.Repo{Name: "repo", Subdir: "/docs/README.md", IsFile: true},
			args: []string{"src", existing},
			want: filepath.Join(existing, "README.md"),
		},
		{
			name: "file, dst is a fresh path -> literal",
			repo: &degit.Repo{Name: "repo", Subdir: "/docs/README.md", IsFile: true},
			args: []string{"src", fresh},
			want: fresh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDestination(tt.repo, tt.args, tt.flat)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDestinationConflict(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "afile")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	missing := filepath.Join(dir, "does-not-exist")

	tests := []struct {
		name    string
		dst     string
		flat    bool
		force   bool
		isFile  bool
		wantErr bool
	}{
		{name: "missing dst -> no conflict", dst: missing},
		{name: "existing dir, no force -> conflict", dst: dir, wantErr: true},
		{name: "existing file, no force -> conflict", dst: file, wantErr: true},
		{name: "existing dir, force -> ok", dst: dir, force: true},
		{name: "existing dir, force, file target -> refuse clobbering dir", dst: dir, force: true, isFile: true, wantErr: true},
		{name: "flat, existing dir, no force -> ok (merge)", dst: dir, flat: true},
		{name: "flat, existing file, no force -> refuse non-dir", dst: file, flat: true, wantErr: true},
		{name: "flat, existing dir, force -> ok (force wins)", dst: dir, flat: true, force: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := destinationConflict(tt.dst, tt.flat, tt.force, tt.isFile)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
