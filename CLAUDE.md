# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`degit` is a Go port of [rich-harris/degit](https://github.com/rich-harris/degit). It downloads a repository tarball (no git history) from GitHub, GitLab, Bitbucket, or Sourcehut and extracts it into a local directory, with an on-disk cache keyed by commit hash.

## Common commands

- Build: `go build -o degit` (or `make release-local` for a goreleaser snapshot)
- Run: `go run . user/repo#ref output-dir`
- Test all: `go test -v ./...`
- Test one: `go test -v -run TestParse ./pkg` (e.g. `TestParse`, `TestClone`)
- Note: `TestClone`, `TestCloneSubdirViaWebURL`, and `TestCloneFileViaWebURL` hit GitHub live (clone `rich-harris/degit` into a `t.TempDir()`). The two web-URL tests pin to a stable tag (`v2.8.5`) for reproducibility. Pinning to a raw commit SHA does **not** work — `getHash` only resolves refs returned by `git ls-remote`, so historical commits not at a branch/tag tip can't be downloaded. All three tests can flake on network or rate limits.
- Release (CI does this on tag push): driven by `.goreleaser.yaml`, which also publishes the `degit.rb` Homebrew formula in this repo

## Architecture

### Command dispatch (`main.go` + `cmd/`)

The binary uses Cobra, but `main.go:setDefaultCommandIfNonePresent` rewrites `os.Args` so that an unrecognized first argument is treated as input to the `clone` subcommand. This is what makes `degit user/repo#ref dst` work without typing `clone`. When adding a new subcommand, ensure it shows up in `cmd.Subcommands()` (Cobra registration in an `init()` is sufficient) — otherwise its name will be intercepted by this shim.

Global flags (`--verbose`, `--force`) live as package-level vars in `cmd/root.go` and are read directly by subcommand `RunE` funcs.

### Core flow (`pkg/`)

`pkg.Clone` → `ParseRepo` → `Repo.Clone`. The package is published as `degit "github.com/qiushiyan/degit/pkg"` (note the import alias).

1. **`parse.go`** — Two-stage parser. HTTPS URLs to known web hosts (`github.com/<u>/<r>/tree|blob/<ref>/<path>`, `raw.githubusercontent.com/...`) go through a structured fast-path (`tryParseWebURL`) that handles the paste-from-browser case and sets `Repo.IsFile` for blob/raw inputs. Everything else (native syntax `u/r#ref`, SSH URLs, `git@...`, GitLab/Bitbucket/Sourcehut hosts) falls through to a single regex. Site detection on the regex path picks the first non-empty capture group and falls back to `github`; the site string is the bare name (`github`, `gitlab`, `bitbucket`, `sourcehut`, `git.sr.ht`). `Ref` defaults to the sentinel `"HEAD"`, resolved by `getHash` later.

2. **`repo.go:Resolve`** — Discovers the commit hash and sets `Repo.Hash` + `Repo.Cached`. `Clone` calls it internally when `Hash` is empty, so library callers that only use `Clone` need not call it. The CLI calls it explicitly first so it can render a pre-action status line (resolved ref or cache-hit hint) before any download begins.

3. **`repo.go:getRefs`** — Shells out to `git ls-remote <url>` to discover refs. **The `git` binary must be on PATH** at runtime; this is the project's only non-Go runtime dependency. `getHash` then resolves the user's ref against the parsed list (HEAD → branch/tag name → commit-hash prefix, min 7 chars).

4. **`repo.go:download`** — Builds a tarball URL whose shape varies per host:
   - GitHub/Sourcehut: `{URL}/archive/{hash}.tar.gz`
   - GitLab: `{URL}/repository/archive.tar.gz?ref={hash}`
   - Bitbucket: `{URL}/get/{hash}.tar.gz`
   When adding a new host, both `parse.go` (host map + domain mapping) and this switch must be updated.

5. **`cache.go`** — Cache lives at `$HOME/.go-degit/{site}/{user}/{name}/`. Each successful download writes `{hash}.tar.gz` plus two sidecar JSON files: `map.json` (ref → hash) and `access.json` (ref → last-access timestamp). When `map.json` shows a ref now resolves to a different hash, the old `{oldHash}.tar.gz` is deleted to avoid unbounded growth. `degit clear` removes the entire tree, or a single repo subtree when given a filter.

6. **`untar.go`** — Extracts the tarball, stripping the archive's top-level directory (`{name}-{hash}/`). Two modes controlled by `isFile`:
   - **Folder mode** (`isFile=false`): if `subdir` is set, entries outside it are skipped and the subdir prefix is stripped; surviving entries land under `dst`. Missing subdir is a silent no-op — the empty-output message in `cmd/clone.go` is the only signal.
   - **File mode** (`isFile=true`): exactly one entry matches `subdir` and its bytes are written to `dst` directly (which is a file path, not a directory). Returns `file not found in repository: X` if the entry is missing.

   `pax_global_header` entries are explicitly skipped in both modes.

### Things to know when editing

- `cmd/clone.go` derives the destination directory from the subdir name (or repo name) when the user omits the second positional arg — keep this behavior in sync with how `untar.go` handles subdir stripping, or output paths will surprise users.
- `Repo.Clone` calls `os.RemoveAll(dst)` when `--force` is set. Don't reorder the "destination exists" check without preserving this.
- `getHash` requires commit-hash refs to be ≥7 chars. Branch and tag matching happen first, so a 6-char branch name still works; only the hash-prefix fallback enforces the minimum.
- **Web URL ref parsing is naive.** `tryParseWebURL` treats the segment immediately after `tree`/`blob` as the ref. Branch names with `/` (e.g. `feat/foo/bar`) will mis-resolve from a github.com web URL — the user gets a clean `could not find ref X for repo Y`. Workaround is native syntax. We deliberately do not disambiguate against the refs list (`getRefs`) because the added complexity isn't worth it for a rare case.
- **File targets use cp-like dst semantics.** `cmd/clone.go:resolveDestination` interprets `dst` based on `Repo.IsFile`: omitted → file's basename; existing-dir → join basename inside; otherwise literal. `Repo.Clone` correspondingly skips its usual `MkdirAll(dst)` and does `MkdirAll(filepath.Dir(dst))` in file mode so `dst` itself can become the file. `--force` overwrites a pre-existing file; pointing a file target at an existing directory always errors (we refuse to clobber a directory with a single file).
- `AlecAivazis/survey/v2` (used in `cmd/clear.go` for the confirmation prompt) was archived by its author in 2023. It still functions, but if a task involves reworking interactive prompts, prefer migrating to `charmbracelet/huh` over extending survey usage.
- **Resolve-then-Clone is idempotent.** `Resolve()` is a no-op when `Hash` is already set, so the CLI's pattern of calling `Resolve()` before `Clone()` is safe. Don't collapse them into a single `Clone`-only call — doing so loses the pre-action status line without any correctness gain.
- **All status/UI output goes to stderr.** `log()`, the progress bar, and every user-facing message use `os.Stderr`. Do not introduce `fmt.Println`/`fmt.Printf` (which write to stdout) without an explicit `os.Stderr` redirect — stdout is reserved for machine-readable output. The TTY guard in `cmd/clone.go` that disables the bar on non-TTY is load-bearing; do not remove it.
