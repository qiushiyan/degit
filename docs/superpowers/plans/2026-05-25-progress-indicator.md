# Progress Indicator and Clone Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a default-on download progress bar plus subtle status lines (resolved ref, cache hit, done line) to `degit clone`, while keeping the `pkg` library UI-free.

**Architecture:** A new `pkg.Progress` interface lets the library report download bytes/total without depending on any UI lib. The CLI implements `Progress` via a small adapter around `schollz/progressbar/v3`. A new `Repo.Resolve()` exposes the hash + cache-hit signal up to the CLI so it can choose the right status line and decide whether to build a bar. All status output moves to stderr.

**Tech Stack:** Go 1.26, Cobra, `schollz/progressbar/v3` (new direct dep), `golang.org/x/term` (already transitive, promoted to direct), `stretchr/testify` for assertions, `net/http/httptest` for offline download tests.

**Spec reference:** `docs/superpowers/specs/2026-05-25-progress-indicator-design.md`

---

## Pre-flight

Run from repo root (`/Users/qiushi/dev/degit`). Verify clean state:

```bash
git status
go test ./...
```

Expected: working tree clean, all tests pass. Three live tests (`TestClone*`) require network — if offline, run `go test -run 'TestParse|TestUntar' ./...` instead.

---

## Task 1: Add `Progress` interface and wire it through `download()`

**Files:**
- Create: `pkg/progress.go`
- Modify: `pkg/repo.go` (add `Progress` field on `Repo`; modify `download()` to call Progress methods)
- Create: `pkg/repo_progress_test.go`

- [ ] **Step 1: Create the `Progress` interface**

Create `pkg/progress.go`:

```go
package degit

import "io"

// Progress is an optional hook for surfacing download progress.
// Implementations receive a copy of downloaded bytes via Write,
// learn the total length via Init (called once before the first Write,
// possibly again on a redirected response), and are notified of the
// end of the download via Finish.
//
// The pkg library never imports a progress bar UI library; consumers
// (e.g. the degit CLI) provide an adapter implementing this interface.
type Progress interface {
	io.Writer
	// Init is called once before bytes start flowing. total is the
	// response Content-Length, or -1 if the server did not advertise one.
	Init(total int64)
	// Finish is called after the last byte is written (success or error).
	Finish()
}
```

- [ ] **Step 2: Add `Progress` field to `Repo`**

In `pkg/repo.go`, locate the `Repo` struct (around line 18) and add a `Progress` field at the end:

```go
type Repo struct {
	Site     string
	User     string
	Name     string
	Ref      string
	URL      string
	SSH      string
	Subdir   string
	IsFile   bool
	Progress Progress // optional; nil = silent (default)
}
```

- [ ] **Step 3: Write failing tests for `Progress` wiring**

Create `pkg/repo_progress_test.go`:

```go
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
	require.Equal(t, int64(13), fp.initTotal, "Init should receive the Content-Length")
	require.Equal(t, 13, fp.bytesWritten, "all body bytes should be forwarded to Progress")
	require.Equal(t, 1, fp.finishCount, "Finish should be called exactly once")

	// file should still contain the full body
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, body, got)
}
```

- [ ] **Step 4: Run the test to verify it fails**

```bash
go test -run TestDownloadCallsProgress ./pkg -v
```

Expected: FAIL — `Init should be called exactly once / Want: 1, Got: 0` (download() does not yet call Progress methods).

- [ ] **Step 5: Wire Progress through `download()`**

In `pkg/repo.go`, locate the `download` method (around line 96) and modify the body so the response copy goes through `Progress` when set:

```go
func (r *Repo) download(dst string, hash string, verbose bool) error {
	folder, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer folder.Close()

	var url string
	switch r.Site {
	case "gitlab":
		url = fmt.Sprintf("%s/repository/archive.tar.gz?ref=%s", r.URL, hash)
	case "bitbucket":
		url = fmt.Sprintf("%s/get/%s.tar.gz", r.URL, hash)
	default:
		url = fmt.Sprintf("%s/archive/%s.tar.gz", r.URL, hash)
	}

	log(verbose, "downloading from", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not find repository %s", r.URL)
	}
	if resp.StatusCode != 200 {
		return r.download(dst, resp.Header.Values("Location")[0], verbose)
	}
	defer resp.Body.Close()

	if r.Progress != nil {
		r.Progress.Init(resp.ContentLength)
		defer r.Progress.Finish()
	}

	var sink io.Writer = folder
	if r.Progress != nil {
		sink = io.MultiWriter(folder, r.Progress)
	}

	_, err = io.Copy(sink, resp.Body)
	return err
}
```

- [ ] **Step 6: Run the test to verify it passes**

```bash
go test -run TestDownloadCallsProgress ./pkg -v
```

Expected: PASS.

- [ ] **Step 7: Run the full pkg test suite to verify no regression**

```bash
go test -run 'TestParse|TestUntar' ./pkg -v
```

Expected: all PASS. (Skipping live-network `TestClone*` to keep this fast; they're covered later.)

- [ ] **Step 8: Commit**

```bash
git add pkg/progress.go pkg/repo.go pkg/repo_progress_test.go
git commit -m "feat(pkg): add Progress interface and wire through download"
```

---

## Task 2: Cover nil-Progress and missing Content-Length cases

**Files:**
- Modify: `pkg/repo_progress_test.go` (append two tests)

- [ ] **Step 1: Add the failing tests**

Append to `pkg/repo_progress_test.go`:

```go
func TestDownloadNilProgressStillWorks(t *testing.T) {
	body := []byte("payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "7")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	repo := newTestRepo(server.URL, nil) // Progress is nil
	dst := filepath.Join(t.TempDir(), "out.tar.gz")

	require.NoError(t, repo.download(dst, "deadbeef", false))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, body, got)
}

func TestDownloadInitWithMinusOneWhenContentLengthMissing(t *testing.T) {
	body := []byte("payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force chunked encoding so Content-Length is absent on the response.
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write(body[:3])
		flusher.Flush()
		_, _ = w.Write(body[3:])
	}))
	defer server.Close()

	fp := &fakeProgress{}
	repo := newTestRepo(server.URL, fp)
	dst := filepath.Join(t.TempDir(), "out.tar.gz")

	require.NoError(t, repo.download(dst, "deadbeef", false))

	require.Equal(t, 1, fp.initCount)
	require.Equal(t, int64(-1), fp.initTotal,
		"missing Content-Length should be surfaced as -1 (Go's http convention)")
	require.Equal(t, len(body), fp.bytesWritten)
}
```

- [ ] **Step 2: Run the new tests**

```bash
go test -run 'TestDownloadNilProgressStillWorks|TestDownloadInitWithMinusOneWhenContentLengthMissing' ./pkg -v
```

Expected: both PASS. The wiring from Task 1 already handles both cases — `Progress != nil` guard covers nil, and `resp.ContentLength` is `-1` when the header is missing (Go stdlib behaviour).

If either fails, the Task 1 wiring is wrong — review the `download()` changes from Task 1, Step 5.

- [ ] **Step 3: Commit**

```bash
git add pkg/repo_progress_test.go
git commit -m "test(pkg): cover nil Progress and missing Content-Length"
```

---

## Task 3: Add `Resolve()`, `Hash`, `Cached` fields; refactor `Clone`

**Files:**
- Modify: `pkg/repo.go`
- Create: `pkg/resolve_test.go`

- [ ] **Step 1: Add `Hash` and `Cached` fields to `Repo`**

In `pkg/repo.go`, extend the struct:

```go
type Repo struct {
	Site     string
	User     string
	Name     string
	Ref      string
	URL      string
	SSH      string
	Subdir   string
	IsFile   bool
	Progress Progress // optional; nil = silent (default)
	Hash     string   // populated by Resolve(); the resolved commit hash
	Cached   bool     // populated by Resolve(); true if the tarball is already in cache
}
```

- [ ] **Step 2: Add the `Resolve` method**

Append to `pkg/repo.go` (place it just above the existing `func (r *Repo) Clone(...)`):

```go
// Resolve discovers the commit hash that r.Ref points to and checks whether
// the resulting tarball is already in the on-disk cache. It is safe to call
// multiple times; subsequent calls are a no-op once Hash is populated.
//
// Clone calls Resolve internally if Hash is empty, so direct library users
// who only call Clone do not need to call Resolve. The CLI calls Resolve
// first so it can print the resolved ref (or a cache-hit hint) before any
// download begins.
func (r *Repo) Resolve() error {
	if r.Hash != "" {
		return nil
	}
	refs, err := r.getRefs()
	if err != nil {
		return err
	}
	hash, err := r.getHash(refs)
	if err != nil {
		return err
	}
	r.Hash = hash

	cached, err := exists(r.getOutputFile(hash))
	if err != nil {
		return err
	}
	r.Cached = cached
	return nil
}
```

- [ ] **Step 3: Refactor `Clone` to use the populated fields**

Replace the body of `Clone` in `pkg/repo.go` (currently lines ~30-94) with this. The pre-existing destination-exists / force logic is unchanged; only the ref+cache resolution is delegated to `Resolve`:

```go
// Clone downloads the repository into the destination
func (r *Repo) Clone(dst string, force bool, verbose bool) error {
	dstExists, err := exists(dst)
	if err != nil {
		return err
	}
	if dstExists {
		if force {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("output location %s already exists", dst)
		}
	}

	if err := r.Resolve(); err != nil {
		return err
	}

	file := r.getOutputFile(r.Hash)

	if !r.Cached {
		if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
			return err
		}
		if err := r.download(file, r.Hash, verbose); err != nil {
			return err
		}
	} else {
		log(verbose, "using cache for", r.URL)
	}

	if err := updateCache(filepath.Dir(file), r.Ref, r.Hash, verbose); err != nil {
		return err
	}

	if r.IsFile {
		err = os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	} else {
		err = os.MkdirAll(dst, os.ModePerm)
	}
	if err != nil {
		return err
	}

	return untar(file, dst, r.Subdir, fmt.Sprintf("%s-%s", r.Name, r.Hash), r.IsFile)
}
```

- [ ] **Step 4: Write a unit test for `Resolve`'s no-op branch**

Resolve's "first-call" branch shells to `git ls-remote`, which we can't run offline without a network dependency. The branch we *can* cover offline is the no-op early return — used by the CLI flow where the CLI calls Resolve once, then Clone calls it again internally and should do nothing. The full-resolution path is exercised end-to-end by the existing live `TestClone*` tests in `pkg/repo_test.go`.

Create `pkg/resolve_test.go`:

```go
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
```

- [ ] **Step 5: Run the new test**

```bash
go test -run TestResolveIsNoOpWhenHashSet ./pkg -v
```

Expected: PASS.

- [ ] **Step 6: Run the parse/untar tests to verify no regression**

```bash
go test -run 'TestParse|TestUntar|TestDownload' ./pkg -v
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/repo.go pkg/resolve_test.go
git commit -m "feat(pkg): add Resolve method with Hash and Cached fields"
```

---

## Task 4: Move pkg verbose output to stderr

**Files:**
- Modify: `pkg/repo.go` (the `log` helper)
- Modify: `pkg/cache.go` (two direct `fmt.Println` / `fmt.Printf` calls)

- [ ] **Step 1: Update the `log` helper**

In `pkg/repo.go`, locate `log` (around line 166):

```go
func log(verbose bool, msg ...any) {
	if verbose {
		fmt.Println(msg...)
	}
}
```

Replace it with:

```go
func log(verbose bool, msg ...any) {
	if verbose {
		fmt.Fprintln(os.Stderr, msg...)
	}
}
```

(`os` is already imported in this file.)

- [ ] **Step 2: Update the direct prints in `pkg/cache.go`**

In `pkg/cache.go`, locate the two verbose lines inside `ClearCache`:

```go
if verbose {
	fmt.Println("no cache found")
}
```
and
```go
if verbose {
	fmt.Printf("no cache found for %s\n", filter)
}
```

Replace them with:

```go
if verbose {
	fmt.Fprintln(os.Stderr, "no cache found")
}
```
and
```go
if verbose {
	fmt.Fprintf(os.Stderr, "no cache found for %s\n", filter)
}
```

Add `"os"` to the import block at the top of `pkg/cache.go` if not already present:

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"
)
```

(It is already imported — confirm before editing.)

- [ ] **Step 3: Verify build and tests**

```bash
go build ./...
go test -run 'TestParse|TestUntar|TestDownload|TestResolve' ./pkg -v
```

Expected: builds clean, tests PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/repo.go pkg/cache.go
git commit -m "refactor(pkg): write verbose output to stderr"
```

---

## Task 5: Add `schollz/progressbar/v3` dependency

**Files:**
- Modify: `go.mod`, `go.sum` (via `go get`)

- [ ] **Step 1: Add the dependency**

```bash
go get github.com/schollz/progressbar/v3@latest
```

Expected output includes `go: added github.com/schollz/progressbar/v3 v3.X.X`.

- [ ] **Step 2: Also promote `golang.org/x/term` to a direct dep**

The CLI's TTY guard (next task batch) will use it. Adding it now means it lands in the `require` (not `require // indirect`) block.

```bash
go get golang.org/x/term@latest
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: builds clean.

- [ ] **Step 4: Confirm `go.mod` shape**

```bash
grep -E 'progressbar|x/term' go.mod
```

Expected: both lines appear in the top-level `require` block (no `// indirect` comments).

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add schollz/progressbar/v3 and promote x/term to direct"
```

---

## Task 6: Implement the CLI progress adapter

**Files:**
- Create: `cmd/progress.go`
- Create: `cmd/progress_test.go`

- [ ] **Step 1: Create the adapter**

Create `cmd/progress.go`:

```go
package cmd

import (
	"io"
	"os"
	"time"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/schollz/progressbar/v3"
)

// cliProgress is the CLI-side implementation of pkg.Progress. It wraps a
// schollz/progressbar bar but defers construction until Init so the bar
// receives the correct total. Write is a no-op until Init runs.
type cliProgress struct {
	desc string
	out  io.Writer
	bar  *progressbar.ProgressBar
}

// Compile-time check that we satisfy the library interface.
var _ degit.Progress = (*cliProgress)(nil)

// newCLIProgress constructs an adapter that renders to stderr.
func newCLIProgress(desc string) *cliProgress {
	return newCLIProgressTo(os.Stderr, desc)
}

// newCLIProgressTo is the testable constructor; production callers use
// newCLIProgress, which targets os.Stderr.
func newCLIProgressTo(w io.Writer, desc string) *cliProgress {
	return &cliProgress{desc: desc, out: w}
}

func (p *cliProgress) Init(total int64) {
	if p.bar == nil {
		p.bar = progressbar.NewOptions64(
			total,
			progressbar.OptionSetDescription(p.desc),
			progressbar.OptionSetWriter(p.out),
			progressbar.OptionShowBytes(true),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetPredictTime(false),
		)
		return
	}
	// Redirect path: download() recursed with a new Content-Length.
	p.bar.ChangeMax64(total)
}

func (p *cliProgress) Write(b []byte) (int, error) {
	if p.bar == nil {
		return len(b), nil
	}
	return p.bar.Write(b)
}

func (p *cliProgress) Finish() {
	if p.bar != nil {
		_ = p.bar.Finish()
	}
}
```

- [ ] **Step 2: Write the adapter smoke test**

Create `cmd/progress_test.go`:

```go
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
```

- [ ] **Step 3: Run the test**

```bash
go test ./cmd -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/progress.go cmd/progress_test.go
git commit -m "feat(cmd): add CLI progress bar adapter for pkg.Progress"
```

---

## Task 7: Add `--no-progress` and `--quiet` / `-q` flags

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add the flag variables and registrations**

In `cmd/root.go`, the file currently exports `Verbose` and `Force` as package-level vars and registers them in `init()`. Add `NoProgress` and `Quiet` next to them.

Replace the whole `cmd/root.go` body below the import block with:

```go
var Verbose bool
var Force bool
var NoProgress bool
var Quiet bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "degit user/repo#ref output-dir",
	Short: "Straightforward project scaffolding",
	Long: `A Go port of the node degit cli https://github.com/rich-harris/degit.

Usage:

	degit user/repo#ref output-dir

This will download a tarball for the repository github.com/user/repo at "ref" locally, and extracts it to output-dir. You can specify subdirectories and use Gitlab and Bitbucket repositories as well. degit also maintains a cache of downloaded tarballs that can be cleared with "degit clear".`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func Subcommands() (commandNames []string) {
	for _, command := range rootCmd.Commands() {
		commandNames = append(commandNames, append(command.Aliases, command.Name())...)
	}
	return
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().
		BoolVarP(&Force, "force", "f", false, "overwrite destination directory")
	rootCmd.PersistentFlags().
		BoolVar(&NoProgress, "no-progress", false, "suppress the download progress bar")
	rootCmd.PersistentFlags().
		BoolVarP(&Quiet, "quiet", "q", false, "suppress all non-error output (mutually exclusive with --verbose)")
	rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceUsage = true
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: builds clean.

- [ ] **Step 3: Smoke-test the flag parsing**

```bash
go run . --help 2>&1 | grep -E -- "--no-progress|--quiet|--verbose"
```

Expected: three lines, one for each flag, in that vicinity.

```bash
go run . -q -v 2>&1 || true
```

Expected: an error similar to `if any flags in the group [quiet verbose] are set none of the others can be; [quiet verbose] were all set`.

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat(cmd): add --no-progress and --quiet flags"
```

---

## Task 8: Add status helpers (`printResolved`, `printCacheHit`, `printDone`)

**Files:**
- Create: `cmd/status.go`
- Create: `cmd/status_test.go`

- [ ] **Step 1: Create the helpers**

Create `cmd/status.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
)

// printResolved announces a download that's about to start. In file mode it
// names the target file rather than showing the short hash, because the
// user usually cares more about which file is being saved than which commit.
func printResolved(w io.Writer, r *degit.Repo) {
	if r.IsFile {
		name := filepath.Base(strings.TrimPrefix(r.Subdir, "/"))
		fmt.Fprintf(w, "↓ %s/%s@%s → %s\n", r.User, r.Name, r.Ref, name)
		return
	}
	fmt.Fprintf(w, "↓ %s/%s@%s (%s)\n", r.User, r.Name, r.Ref, shortHash(r.Hash))
}

// printCacheHit announces that the tarball was already on disk.
func printCacheHit(w io.Writer, r *degit.Repo) {
	fmt.Fprintf(w, "↪ using cache %s/%s@%s (%s)\n",
		r.User, r.Name, r.Ref, shortHash(r.Hash))
}

// printDone confirms successful extraction.
func printDone(w io.Writer, r *degit.Repo, dst string) {
	if r.IsFile {
		fmt.Fprintf(w, "✓ saved to %s\n", dst)
		return
	}
	fmt.Fprintf(w, "✓ extracted to %s\n", dst)
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
```

(The escape sequences `↓`, `↪`, `✓` are the down-arrow, hooked-arrow, and check-mark glyphs. Using escapes keeps the source ASCII-clean.)

- [ ] **Step 2: Write tests for the formatting**

Create `cmd/status_test.go`:

```go
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
```

- [ ] **Step 3: Run the tests**

```bash
go test -run 'TestPrint|TestShortHash' ./cmd -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/status.go cmd/status_test.go
git commit -m "feat(cmd): add status line helpers for clone feedback"
```

---

## Task 9: Wire progress + status into `cmd/clone.go`

**Files:**
- Modify: `cmd/clone.go`

- [ ] **Step 1: Replace the `RunE` body**

Open `cmd/clone.go`. The file currently has:

- An import block
- A `cloneCmd` `var` with a `RunE` closure
- A `resolveDestination` helper
- An `init()` that calls `rootCmd.AddCommand(cloneCmd)`

Replace the import block with:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)
```

Replace the `RunE` closure (everything between `RunE: func(cmd *cobra.Command, args []string) error {` and its matching `},` before `resolveDestination`) with:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := degit.ParseRepo(args[0])
		if err != nil {
			return err
		}

		dst := resolveDestination(repo, args)

		if stat, err := os.Stat(dst); err == nil {
			if !Force {
				return fmt.Errorf("destination `%s` already exists, use --force to overwrite", dst)
			}
			if repo.IsFile && stat.IsDir() {
				return fmt.Errorf("destination `%s` is a directory; refusing to overwrite with a file", dst)
			}
		}

		if Verbose {
			fmt.Fprintf(os.Stderr, "Cloning `%s` into `%s`\n", repo.URL, dst)
		}

		if err := repo.Resolve(); err != nil {
			return err
		}

		if !Quiet {
			if repo.Cached {
				printCacheHit(os.Stderr, repo)
			} else {
				printResolved(os.Stderr, repo)
			}
		}

		if !Quiet && !NoProgress && !repo.Cached && term.IsTerminal(int(os.Stderr.Fd())) {
			repo.Progress = newCLIProgress("downloading")
		}

		if err := repo.Clone(dst, Force, Verbose); err != nil {
			return err
		}

		if !Quiet {
			printDone(os.Stderr, repo, dst)
		}

		if repo.IsFile {
			return nil
		}

		entries, err := os.ReadDir(dst)
		if err != nil {
			return err
		}
		if len(entries) == 0 && !Quiet {
			fmt.Fprintln(
				os.Stderr,
				"Output directory is empty, you might have specified an non-existing subfolder in the repository",
			)
		}
		return nil
	},
```

Leave `resolveDestination` and `init()` unchanged.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: builds clean. If you see `term` import errors, confirm Task 5 promoted `golang.org/x/term` to a direct dep.

- [ ] **Step 3: Run all unit tests**

```bash
go test -run 'TestParse|TestUntar|TestDownload|TestResolve|TestPrint|TestShortHash|TestCLIProgress' ./...
```

Expected: all PASS.

- [ ] **Step 4: Manual smoke test — fresh download**

```bash
go run . rich-harris/degit /tmp/degit-smoke 2>&1
```

Expected output on a TTY:

```
↓ rich-harris/degit@HEAD (<short hash>)
downloading  ... [bar redraws] ...
✓ extracted to /tmp/degit-smoke
```

(Bar may be very fast on a small repo — that's fine.)

Clean up: `rm -rf /tmp/degit-smoke`.

- [ ] **Step 5: Manual smoke test — cache hit**

Re-run the same command (cache should now exist):

```bash
go run . rich-harris/degit /tmp/degit-smoke 2>&1
```

Expected: no bar, instead:

```
↪ using cache rich-harris/degit@HEAD (<short hash>)
✓ extracted to /tmp/degit-smoke
```

Clean up: `rm -rf /tmp/degit-smoke`.

- [ ] **Step 6: Manual smoke test — `--no-progress`**

```bash
rm -rf ~/.go-degit/github/rich-harris  # force a fresh download
go run . --no-progress rich-harris/degit /tmp/degit-smoke 2>&1
```

Expected:

```
↓ rich-harris/degit@HEAD (<short hash>)
✓ extracted to /tmp/degit-smoke
```

(No bar; status lines remain.) Clean up: `rm -rf /tmp/degit-smoke`.

- [ ] **Step 7: Manual smoke test — `--quiet`**

```bash
rm -rf ~/.go-degit/github/rich-harris
go run . -q rich-harris/degit /tmp/degit-smoke 2>&1
```

Expected: zero output on success. Clean up: `rm -rf /tmp/degit-smoke`.

- [ ] **Step 8: Manual smoke test — non-TTY (piped stderr)**

```bash
rm -rf ~/.go-degit/github/rich-harris
go run . rich-harris/degit /tmp/degit-smoke 2>&1 | cat
```

Expected: status lines visible; no `\r` carriage-return garbage from the bar (the TTY guard suppresses it when stderr is piped).

Clean up: `rm -rf /tmp/degit-smoke`.

- [ ] **Step 9: Commit**

```bash
git add cmd/clone.go
git commit -m "feat(cmd): wire progress bar and status lines into clone"
```

---

## Task 10: Move `cmd/clear.go` output to stderr

**Files:**
- Modify: `cmd/clear.go`

- [ ] **Step 1: Update the two direct `fmt` calls**

In `cmd/clear.go`, locate these two lines (around lines 29 and 60):

```go
fmt.Println("no cache found, skipping")
```
and
```go
fmt.Printf("no cache found for %s\n", filter)
```

Replace them with:

```go
fmt.Fprintln(os.Stderr, "no cache found, skipping")
```
and
```go
fmt.Fprintf(os.Stderr, "no cache found for %s\n", filter)
```

(`os` is already imported.)

- [ ] **Step 2: Verify build and tests**

```bash
go build ./... && go test ./cmd ./pkg -run 'TestParse|TestUntar|TestDownload|TestResolve|TestPrint|TestShortHash|TestCLIProgress' -v
```

Expected: builds clean, tests PASS.

- [ ] **Step 3: Manual smoke test**

```bash
rm -rf ~/.go-degit
go run . clear 2>&1
```

Expected (when typed `n` at prompt or otherwise no cache present): `no cache found, skipping` on stderr.

- [ ] **Step 4: Commit**

```bash
git add cmd/clear.go
git commit -m "refactor(cmd): write clear command status to stderr"
```

---

## Task 11: Final integration check

**Files:** none — verification only.

- [ ] **Step 1: Run the full test suite (offline subset)**

```bash
go test -run 'TestParse|TestUntar|TestDownload|TestResolve|TestPrint|TestShortHash|TestCLIProgress' ./... -v
```

Expected: all PASS.

- [ ] **Step 2: Run the live network tests**

```bash
go test -v ./pkg -run 'TestClone'
```

Expected: `TestClone`, `TestCloneSubdirViaWebURL`, `TestCloneFileViaWebURL` all PASS. These can flake on rate limits or transient network failures; one retry is acceptable. If `TestCloneSubdirViaWebURL` or `TestCloneFileViaWebURL` fails on a "could not find ref" error, the upstream pinned tag (`v2.8.5`) has moved or been deleted — out of scope for this plan, file a separate issue.

- [ ] **Step 3: Build the release binary**

```bash
go build -o degit
./degit --help 2>&1 | grep -E -- "no-progress|quiet"
rm degit
```

Expected: both flag lines appear in `--help` output.

- [ ] **Step 4: Confirm git history is clean**

```bash
git log --oneline main...HEAD
```

Expected: ten commits in the order they were created (Tasks 1–10), each with a focused message. The Co-Authored-By trailer is not required for this work; commit messages can be plain `feat:` / `refactor:` / `test:` / `chore:` prefixes per the existing project style (see `git log --oneline -20` for tone).

- [ ] **Step 5: Self-review against the spec**

Open `docs/superpowers/specs/2026-05-25-progress-indicator-design.md` side-by-side with `git diff main...HEAD --stat` and confirm:

- [ ] `Progress` interface lives in `pkg/progress.go` (Task 1)
- [ ] `Repo.Progress`, `Hash`, `Cached` fields exist (Tasks 1, 3)
- [ ] `Repo.Resolve()` exposed (Task 3)
- [ ] `Repo.Clone` calls `Resolve` internally and reuses populated fields (Task 3)
- [ ] `cmd/progress.go` has the schollz adapter (Task 6)
- [ ] `cmd/status.go` has the three status helpers (Task 8)
- [ ] `--no-progress` and `--quiet` flags wired with mutex against `--verbose` (Task 7)
- [ ] All status output goes to stderr (Tasks 4, 9, 10)
- [ ] Bar TTY-guarded via `golang.org/x/term.IsTerminal` (Task 9)
- [ ] All four manual smoke tests in Task 9 produced the expected behaviour

If anything is missing, raise it before declaring the plan complete.

---

## Out of scope (recorded; do not implement here)

- A `--plain` flag for ASCII-only output.
- Time-taken display in the done line.
- Replacing the `AlecAivazis/survey/v2` dependency in `cmd/clear.go`.
- Progress for the `git ls-remote` phase.
