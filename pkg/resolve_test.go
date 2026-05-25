package degit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveIsNoOpWhenHashSet(t *testing.T) {
	repo := &Repo{
		Site: "github",
		User: "u",
		Name: "r",
		Ref:  "main",
		Hash: "abc1234deadbeef", // pre-populated
	}
	require.NoError(t, repo.Resolve(), "Resolve must not error when Hash is already set")
	require.Equal(t, "abc1234deadbeef", repo.Hash, "Hash must not change")
	require.False(t, repo.Cached, "Cached must stay at zero-value when Resolve is a no-op")
}
