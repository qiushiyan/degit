package degit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// pinnedTag is a stable tag on rich-harris/degit. Tags don't move, so the
// tests stay reproducible across upstream churn. We use a pre-3.0 release
// because v3 converted to TypeScript (src/*.ts) and moved help.md under
// assets/, which would invalidate the assertions below.
//
// Pinning to a raw commit SHA does NOT work: degit's getHash only resolves
// refs returned by `git ls-remote`, so historical commits that aren't at a
// branch/tag tip can't be downloaded.
const pinnedTag = "v2.8.5"

func TestClone(t *testing.T) {
	repo, err := ParseRepo("github.com/rich-harris/degit")
	require.NoError(t, err)

	err = repo.Clone(t.TempDir(), true, true)
	require.NoError(t, err)
}

func TestCloneSubdirViaWebURL(t *testing.T) {
	src := "https://github.com/rich-harris/degit/tree/" + pinnedTag + "/src"
	repo, err := ParseRepo(src)
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, repo.Clone(dir, true, false))

	for _, name := range []string{"bin.js", "index.js", "utils.js"} {
		info, err := os.Stat(filepath.Join(dir, name))
		require.NoErrorf(t, err, "expected %s to be extracted", name)
		require.Falsef(t, info.IsDir(), "%s should be a file", name)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); !os.IsNotExist(err) {
		t.Errorf("README.md should not be extracted when subdir=/src")
	}
}

func TestCloneFileViaWebURL(t *testing.T) {
	src := "https://github.com/rich-harris/degit/blob/" + pinnedTag + "/help.md"
	repo, err := ParseRepo(src)
	require.NoError(t, err)

	dst := filepath.Join(t.TempDir(), "out.md")
	require.NoError(t, repo.Clone(dst, true, false))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	require.False(t, info.IsDir(), "dst must be a file, not a directory")
	require.Greater(t, info.Size(), int64(0))

	body, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Contains(t, strings.ToLower(string(body)), "degit",
		"help.md content should mention 'degit'")
}
