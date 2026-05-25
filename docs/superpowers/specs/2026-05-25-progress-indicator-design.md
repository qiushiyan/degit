# Progress indicator and clone feedback — design

- **Date:** 2026-05-25
- **Status:** Draft — pending user review
- **Scope:** CLI-only UX changes; `pkg` library stays headless

## Summary

Add a default-on download progress bar plus a small set of subtle status lines (resolved ref, cache hint, done line) so users get clear, non-intrusive feedback during `degit clone`. The `pkg` library stays free of UI concerns by exposing a small `Progress` interface that the CLI implements via a `schollz/progressbar/v3` adapter.

## Motivation

Today degit is almost completely silent by default. The user has no visibility into:

- Whether a download is in flight or stalled
- Which commit was actually resolved from a branch/tag/HEAD
- Whether the result came from cache (fast) or the network (slower)
- Whether the operation finished, or just dropped back to the prompt early

All existing status lines (`Cloning ...`, `using cache for ...`, `downloading from ...`) only print under `--verbose`. The default behaviour should give users enough signal to trust the tool without forcing them to opt in to noise.

## Goals

1. Show a percentage progress bar during the tarball download.
2. Show a resolved-ref line before the download (or a cache-hit line if no download).
3. Show a "done" confirmation line after extraction.
4. Keep all of this subtle and to-the-point — single short lines, no banners.
5. Auto-disable the bar on non-TTY output (CI, piped stderr) without user action.
6. Keep `pkg` headless: no UI deps leak into the library; the existing exported API (`Clone`, `ParseRepo`, `Repo.Clone`) remains backwards-compatible.

## Non-goals

- Progress for the `git ls-remote` phase (typically <1s; not worth instrumenting).
- Multi-bar UI for parallel downloads (degit runs one download at a time).
- Throughput tuning or HTTP changes; this spec is purely UI/UX.
- Restyling the `degit clear` interactive flow.

## User-visible behaviour

All status output goes to **stderr** (the current `fmt.Println` to stdout is moved as part of this change — see "Stream consistency" below).

### Default (TTY, fresh download)

```
↓ user/repo@main (a1b2c3d)
downloading  42% [==========>            ] (1.2/2.9 MB, 3.4 MB/s)
✓ extracted to ./dst
```

The bar redraws in place and is cleared from the terminal when download completes (`schollz/progressbar`'s `OptionClearOnFinish`). The "✓ extracted" line is the only post-download artifact left on screen.

### Default (cache hit)

```
↪ using cache user/repo@main (a1b2c3d)
✓ extracted to ./dst
```

No progress bar because no network fetch happens.

### File mode (single-file via `blob/` or `raw.githubusercontent.com`)

```
↓ user/repo@main → README.md
downloading  42% [==========>            ] (12.3/29.0 KB, 110 KB/s)
✓ saved to ./README.md
```

### `--no-progress` (TTY)

Suppresses **only the bar**. The status lines remain:

```
↓ user/repo@main (a1b2c3d)
✓ extracted to ./dst
```

### `--quiet` / `-q` (new flag)

Suppresses all status output (bar, status lines, the existing empty-dir warning). Errors still go to stderr. Mutually exclusive with `--verbose`.

```
(no output on success)
```

### `--verbose` / `-v` (unchanged)

Adds the existing chatty log lines on top of the default UX:

```
Cloning `https://github.com/user/repo` into `./dst`
↓ user/repo@main (a1b2c3d)
downloading from https://github.com/user/repo/archive/a1b2c3d.tar.gz
downloading  42% [==========>            ] (1.2/2.9 MB, 3.4 MB/s)
✓ extracted to ./dst
```

`-v` and `--no-progress` compose: verbose without the bar.

### Non-TTY (piped, CI)

The bar is suppressed automatically — no `\r` redraws end up in logs. Status lines still print (they are plain newlines). To silence everything pass `--quiet`.

```
↓ user/repo@main (a1b2c3d)
✓ extracted to ./dst
```

## Architecture

### Library: `pkg.Progress` interface (new)

```go
// Progress is an optional hook for surfacing download progress.
// Implementations receive a copy of downloaded bytes via Write,
// learn the total length via Init, and are notified of the end
// (success or error) via Finish.
//
// pkg never imports a progress bar library; the CLI provides an
// adapter that satisfies this interface.
type Progress interface {
    io.Writer
    Init(total int64) // -1 if Content-Length is unknown
    Finish()
}
```

This is the only new exported symbol in `pkg`. It is small, stable, and decouples the library from any UI library choice.

### Library: `Repo.Progress` field (new optional field)

```go
type Repo struct {
    // ... existing fields
    Progress Progress // optional; nil = silent (default and current behavior)
}
```

`ParseRepo` does not set this; it remains nil for direct library consumers. The CLI sets it after parsing when a TTY is detected and `--no-progress` was not passed.

### Library: `Repo.download` change

The download loop in `pkg/repo.go` becomes:

```go
if r.Progress != nil {
    r.Progress.Init(resp.ContentLength)
    defer r.Progress.Finish()
}

var sink io.Writer = folder
if r.Progress != nil {
    sink = io.MultiWriter(folder, r.Progress)
}

_, err = io.Copy(sink, resp.Body)
```

The redirect path (`r.download` recurses on non-200/non-≥400 responses) re-enters `download`, which re-calls `Init` with the redirected response's `Content-Length`. Adapter must tolerate `Init` being called more than once.

### CLI: progress adapter

A new file `cmd/progress.go` defines a small adapter wrapping `schollz/progressbar/v3`:

```go
type cliProgress struct {
    bar  *progressbar.ProgressBar
    desc string
}

func newCLIProgress(desc string) *cliProgress { ... }

func (p *cliProgress) Init(total int64) {
    if p.bar == nil {
        p.bar = progressbar.NewOptions64(
            total,
            progressbar.OptionSetDescription(p.desc),
            progressbar.OptionSetWriter(os.Stderr),
            progressbar.OptionShowBytes(true),
            progressbar.OptionThrottle(65*time.Millisecond),
            progressbar.OptionClearOnFinish(),
            progressbar.OptionSetPredictTime(false), // we show rate, not ETA
        )
    } else {
        p.bar.ChangeMax64(total) // redirect path
    }
}

func (p *cliProgress) Write(b []byte) (int, error) { return p.bar.Write(b) }
func (p *cliProgress) Finish()                     { _ = p.bar.Finish() }
```

This is the only file in the project that imports `schollz/progressbar/v3`.

### CLI: status helpers

A small new helper in `cmd/clone.go` (or a sibling `cmd/status.go`) prints the three status lines to stderr:

```go
func printResolved(repo *degit.Repo, hash string)   // "↓ user/repo@main (a1b2c3d)"
func printCacheHit(repo *degit.Repo, hash string)   // "↪ using cache ..."
func printDone(repo *degit.Repo, dst string)        // "✓ extracted to ./dst" / "✓ saved to ./X"
```

These respect the `Quiet` flag (see below) and are no-ops when it's set.

### Resolving the ref before download (new `Repo.Resolve`)

Printing the resolved-ref line *before* the bar requires the hash before `download` starts. It also lets us decide whether to even construct the bar (cache hit → no bar). We expose one new method on `Repo`:

```go
// Resolve performs ref discovery (git ls-remote + hash matching) and detects
// whether the resulting tarball is already in the local cache. It populates
// r.Hash and r.Cached so a subsequent r.Clone(...) can reuse them without
// re-resolving.
func (r *Repo) Resolve() error
```

Two new fields on `Repo` back it:

```go
Hash    string // populated by Resolve; empty before
Cached  bool   // populated by Resolve; true if the tarball already exists in cache
```

`Repo.Clone` is refactored to use `r.Hash`/`r.Cached` when set, and to call `r.Resolve()` itself when they are not (so existing direct callers of `Clone` keep working unchanged).

The CLI flow becomes:

```go
if err := repo.Resolve(); err != nil { return err }
if repo.Cached {
    printCacheHit(repo)
} else {
    printResolved(repo)
    if !Quiet && !NoProgress && term.IsTerminal(int(os.Stderr.Fd())) {
        repo.Progress = newCLIProgress("downloading")
    }
}
if err := repo.Clone(dst, Force, Verbose); err != nil { return err }
printDone(repo, dst)
```

Exports added to `pkg`: the `Progress` interface, the `Resolve` method, and the `Progress` / `Hash` / `Cached` fields on `Repo`. The existing `Clone(dst, force, verbose)` and `ParseRepo` signatures are unchanged.

### Stream consistency

All status / UI output (existing verbose lines, new status lines, progress bar) moves to **stderr**. Stdout is left empty on success. Rationale:

- Stderr can be a TTY independently of stdout, so the bar works under `... | tee log` and similar.
- Redirecting stdout (`degit ... > log.txt`) no longer pollutes the file with `\r` garbage.
- Matches `curl`, `wget`, `git`, `npm` conventions.

The empty-dir warning (`Output directory is empty, ...`) also moves to stderr.

## Flag wiring

Two new global flags in `cmd/root.go`:

```go
var NoProgress bool   // --no-progress
var Quiet      bool   // --quiet, -q (mutually exclusive with --verbose)
```

Registration:

```go
rootCmd.PersistentFlags().BoolVar(&NoProgress, "no-progress", false,
    "suppress the download progress bar")
rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false,
    "suppress all non-error output (mutually exclusive with --verbose)")
rootCmd.MarkFlagsMutuallyExclusive("quiet", "verbose")
```

`-v` short form stays bound to `--verbose`. `-q` is the short form for `--quiet`. `--no-progress` has no short form (it's a rare opt-out).

Resolution order inside `cmd/clone.go`:

1. If `Quiet`: no status lines, no bar.
2. Else: status lines always print to stderr.
3. Bar only constructed if `!NoProgress && !Quiet && !cached && term.IsTerminal(int(os.Stderr.Fd()))`.

## Edge cases

| Case | Behaviour |
|---|---|
| `Content-Length` missing/`-1` | Bar runs in spinner mode (schollz auto-handles); rate still shows. |
| HTTP redirect (`r.download` recurses) | Adapter's `Init` called again with new `Content-Length`; `bar.ChangeMax64` updates the total. |
| Cache hit | No bar shown; only the `↪ using cache ...` and `✓ extracted ...` lines. |
| Non-TTY stderr (CI / piped) | Bar skipped at CLI guard; status lines still print (plain newlines, safe in logs). |
| `--quiet` | Total silence on success path; errors still print. |
| `--quiet` + `--verbose` | Cobra rejects at parse time via `MarkFlagsMutuallyExclusive`. |
| `--verbose` + `--no-progress` | Verbose logs print; no bar; status lines print. |
| Ctrl-C mid-download | `progressbar`'s `OptionClearOnFinish` runs in our `Finish()` deferred from `download`; if interrupted by signal mid-write, the next shell prompt may land on the bar line (acceptable; matches curl). |
| Single-file mode | Same UX with `→ <filename>` in the resolved line and `✓ saved to ./X` in the done line. |
| File-mode dst is an existing dir | Existing error path unchanged; no status lines printed. |
| `--force` overwrite | Status lines unchanged; the bar appears once `download` is called. |

## Testing

### New unit tests in `pkg`

`pkg/repo_progress_test.go` — offline, using `httptest.NewServer`:

- Fake `Progress` records `Init(total)` calls and accumulates bytes via `Write`. Assert:
  - `Init` is called exactly once when no redirect.
  - Total argument matches the response's `Content-Length`.
  - Sum of `Write` byte counts equals the body length.
  - `Finish` is called.
- Same test with `Content-Length` header omitted: `Init` called with `-1`.
- A test where `Progress` is nil: download succeeds, no panic, file contents intact (regression guard).

### New unit test in `cmd` (lightweight)

`cmd/progress_test.go`:

- Construct `newCLIProgress("downloading")` with a `bytes.Buffer` swap on `os.Stderr` is too fragile; instead expose the writer choice via an internal constructor (`newCLIProgressTo(w io.Writer, desc string)`) and assert that `Write` forwards bytes (via the bar) without erroring. Cosmetic correctness of the rendered output is not tested.

### Existing tests

`TestClone`, `TestCloneSubdirViaWebURL`, `TestCloneFileViaWebURL` are unaffected — they do not set `Progress` and exercise the silent path. They continue to pass without modification.

## Dependencies

Adds:

- `github.com/schollz/progressbar/v3` — direct.

Transitive: `chengxilo/virtualterm`, `k0kubun/go-ansi`, `mitchellh/colorstring`, `rivo/uniseg`, `golang.org/x/term`, `mattn/go-isatty`, `mattn/go-runewidth`, `golang.org/x/sys`. All pure-Go, small, MIT-compatible licences. Static binary build under goreleaser is unaffected.

`golang.org/x/term` becomes a direct dep (currently transitive) for the TTY guard in `cmd/clone.go`.

## Out of scope (possible follow-ups)

- A `--plain` flag forcing ASCII-only output (no `↓ ↪ ✓` glyphs). Defer until someone reports a terminal that mangles them.
- Time-taken display in the done line (`✓ extracted to ./dst in 212ms`). Could be added later under `-v`.
- Refactor `cmd/clear.go` interactive flow off the archived `survey` library (CLAUDE.md flags this separately).
- Progress for `git ls-remote`.

## Open questions

None at spec-writing time. Adapter signature, flag names, glyph choice, and stderr migration are all decided above.
