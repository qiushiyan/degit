package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIProgressForwardsBytes(t *testing.T) {
	var buf bytes.Buffer
	p := newCLIProgressTo(&buf, "downloading")

	// Write before Init must be a safe no-op that consumes the bytes.
	n, err := p.Write([]byte("ignored"))
	require.NoError(t, err)
	require.Equal(t, len("ignored"), n)

	p.Init(100)

	n, err = p.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)

	// Redirect path: Init again should not panic.
	p.Init(200)

	p.Finish()

	// We don't assert on buf's exact contents (schollz's exact escape
	// sequences are an implementation detail), but it should be non-empty
	// once the bar has rendered.
	require.NotEmpty(t, buf.String(), "bar should have written something to the buffer")
}
