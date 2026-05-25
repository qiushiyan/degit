package degit

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeProgress struct {
	initCount    int
	initTotal    int64
	bytesWritten int
	finishCount  int
}

func (f *fakeProgress) Write(p []byte) (int, error) {
	f.bytesWritten += len(p)
	return len(p), nil
}

func (f *fakeProgress) Init(total int64) {
	f.initCount++
	f.initTotal = total
}

func (f *fakeProgress) Finish() {
	f.finishCount++
}

// newTestRepo builds a Repo pointed at the given test server. We use
// Site="github" so download() takes the default {URL}/archive/{hash}.tar.gz
// branch; the httptest server matches any path.
func newTestRepo(serverURL string, p Progress) *Repo {
	return &Repo{
		Site:     "github",
		User:     "u",
		Name:     "r",
		URL:      serverURL,
		Progress: p,
	}
}

func TestDownloadCallsProgress(t *testing.T) {
	body := []byte("hello, world!")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "13")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	fp := &fakeProgress{}
	repo := newTestRepo(server.URL, fp)
	dst := filepath.Join(t.TempDir(), "out.tar.gz")

	require.NoError(t, repo.download(dst, "deadbeef", false))

	require.Equal(t, 1, fp.initCount, "Init should be called exactly once")
	require.Equal(t, int64(len(body)), fp.initTotal, "Init should receive the Content-Length")
	require.Equal(t, len(body), fp.bytesWritten, "all body bytes should be forwarded to Progress")
	require.Equal(t, 1, fp.finishCount, "Finish should be called exactly once")

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, body, got)
}
