package degit

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClone(t *testing.T) {
	repo, err := ParseRepo("github.com/rich-harris/degit")
	require.NoError(t, err)

	dir := os.TempDir() + "/degit"
	defer os.Remove(dir)
	err = repo.Clone(dir, true, true)
	require.NoError(t, err)
}
