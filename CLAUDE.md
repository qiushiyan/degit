# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`degit` is a Go port of [rich-harris/degit](https://github.com/rich-harris/degit). It downloads a repository tarball (no git history) from GitHub, GitLab, Bitbucket, or Sourcehut and extracts it into a local directory, with an on-disk cache keyed by commit hash.

## Common commands

- Build: `go build -o degit` (or `make release-local` for a goreleaser snapshot)
- Run: `go run . user/repo#ref output-dir`
- Test all: `go test -v ./...`
- Test one: `go test -v -run TestParse ./pkg` (e.g. `TestParse`, `TestClone`)
- Note: `TestClone` hits GitHub live (clones `rich-harris/degit` into `$TMPDIR/degit`). It needs network and can flake on rate limits / outages — not a test-environment issue.
- Release (CI does this on tag push): driven by `.goreleaser.yaml`, which also publishes the `degit.rb` Homebrew formula in this repo

## Architecture

### Command dispatch (`main.go` + `cmd/`)

The binary uses Cobra, but `main.go:setDefaultCommandIfNonePresent` rewrites `os.Args` so that an unrecognized first argument is treated as input to the `clone` subcommand. This is what makes `degit user/repo#ref dst` work without typing `clone`. When adding a new subcommand, ensure it shows up in `cmd.Subcommands()` (Cobra registration in an `init()` is sufficient) — otherwise its name will be intercepted by this shim.

Global flags (`--verbose`, `--force`) live as package-level vars in `cmd/root.go` and are read directly by subcommand `RunE` funcs.

### Core flow (`pkg/`)

`pkg.Clone` → `ParseRepo` → `Repo.Clone`. The package is published as `degit "github.com/qiushiyan/degit/pkg"` (note the import alias).

1. **`parse.go`** — One regex parses every supported URL shape (`user/repo`, `host.com/user/repo/subdir#ref`, `git@host:user/repo`, etc.). Site detection picks the first non-empty capture group and falls back to `github`. The site string is the bare name (`github`, `gitlab`, `bitbucket`, `sourcehut`, `git.sr.ht`) — the full domain is reconstructed when building the URL. `Ref` defaults to the sentinel `"HEAD"`, which `getHash` resolves to the actual HEAD commit.

2. **`repo.go:getRefs`** — Shells out to `git ls-remote <url>` to discover refs. **The `git` binary must be on PATH** at runtime; this is the project's only non-Go runtime dependency. `getHash` then resolves the user's ref against the parsed list (HEAD → branch/tag name → commit-hash prefix, min 7 chars).

3. **`repo.go:download`** — Builds a tarball URL whose shape varies per host:
   - GitHub/Sourcehut: `{URL}/archive/{hash}.tar.gz`
   - GitLab: `{URL}/repository/archive.tar.gz?ref={hash}`
   - Bitbucket: `{URL}/get/{hash}.tar.gz`
   When adding a new host, both `parse.go` (host map + domain mapping) and this switch must be updated.

4. **`cache.go`** — Cache lives at `$HOME/.go-degit/{site}/{user}/{name}/`. Each successful download writes `{hash}.tar.gz` plus two sidecar JSON files: `map.json` (ref → hash) and `access.json` (ref → last-access timestamp). When `map.json` shows a ref now resolves to a different hash, the old `{oldHash}.tar.gz` is deleted to avoid unbounded growth. `degit clear` removes the entire tree, or a single repo subtree when given a filter.

5. **`untar.go`** — Extracts the tarball, stripping the archive's top-level directory (`{name}-{hash}/`). If `Repo.Subdir` is set, entries outside that subdir are skipped and the subdir prefix is also stripped, so the subdir's contents land directly in `dst`. `pax_global_header` entries are explicitly skipped.

### Things to know when editing

- `cmd/clone.go` derives the destination directory from the subdir name (or repo name) when the user omits the second positional arg — keep this behavior in sync with how `untar.go` handles subdir stripping, or output paths will surprise users.
- `Repo.Clone` calls `os.RemoveAll(dst)` when `--force` is set. Don't reorder the "destination exists" check without preserving this.
- `getHash` requires commit-hash refs to be ≥7 chars. Branch and tag matching happen first, so a 6-char branch name still works; only the hash-prefix fallback enforces the minimum.
- `AlecAivazis/survey/v2` (used in `cmd/clear.go` for the confirmation prompt) was archived by its author in 2023. It still functions, but if a task involves reworking interactive prompts, prefer migrating to `charmbracelet/huh` over extending survey usage.
