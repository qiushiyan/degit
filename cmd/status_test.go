package cmd

import (
	"bytes"
	"testing"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/stretchr/testify/require"
)

func TestPrintResolvedFolder(t *testing.T) {
	var buf bytes.Buffer
	r := &degit.Repo{User: "u", Name: "r", Ref: "main", Hash: "abc1234deadbeef"}
	printResolved(&buf, r)
	require.Equal(t, "↓ u/r@main (abc1234)\n", buf.String())
}

func TestPrintResolvedFile(t *testing.T) {
	var buf bytes.Buffer
	r := &degit.Repo{
		User: "u", Name: "r", Ref: "main",
		Hash: "abc1234deadbeef", Subdir: "/docs/README.md", IsFile: true,
	}
	printResolved(&buf, r)
	require.Equal(t, "↓ u/r@main → README.md\n", buf.String())
}

func TestPrintCacheHit(t *testing.T) {
	var buf bytes.Buffer
	r := &degit.Repo{User: "u", Name: "r", Ref: "main", Hash: "abc1234deadbeef"}
	printCacheHit(&buf, r)
	require.Equal(t, "↪ using cache u/r@main (abc1234)\n", buf.String())
}

func TestPrintDoneFolder(t *testing.T) {
	var buf bytes.Buffer
	r := &degit.Repo{User: "u", Name: "r"}
	printDone(&buf, r, "./dst")
	require.Equal(t, "✓ extracted to ./dst\n", buf.String())
}

func TestPrintDoneFile(t *testing.T) {
	var buf bytes.Buffer
	r := &degit.Repo{User: "u", Name: "r", IsFile: true}
	printDone(&buf, r, "./README.md")
	require.Equal(t, "✓ saved to ./README.md\n", buf.String())
}

func TestShortHashUnderEight(t *testing.T) {
	require.Equal(t, "abc123", shortHash("abc123"))
	require.Equal(t, "abc1234", shortHash("abc1234deadbeef"))
}
