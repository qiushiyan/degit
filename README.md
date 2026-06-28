# go-degit

Go port of [degit](https://github.com/rich-harris/degit).

Usage

```bash
degit user/repo#ref output-dir
```

This downloads the GitHub repository `https://github.com/user/repo` at `ref` (a branch, tag, or commit hash) and extracts it locally. If `ref` is omitted, the default branch is used. You can also target subdirectories and single files, and use GitLab, Bitbucket, and Sourcehut repositories. degit maintains a cache to save downloads and keep refs updated.

### Destination

The destination follows `cp -r` / `git clone` conventions, so the `output-dir`
argument behaves the way muscle memory expects:

```bash
# omitted        -> a folder named after the repo (or subdir) in the cwd
degit user/repo                 # -> ./repo/

# existing dir   -> a named subfolder is created inside it
degit user/repo ~/code/         # -> ~/code/repo/

# anything else  -> treated as the literal target (rename)
degit user/repo my-app          # -> ./my-app/
```

Subdirectories and single files work the same way. You can paste a GitHub
browser URL directly:

```bash
# a subdirectory -> ~/.claude/skills/grill-me/
degit https://github.com/mattpocock/skills/tree/main/skills/productivity/grill-me ~/.claude/skills/

# a single file  -> ./SKILL.md
degit https://github.com/mattpocock/skills/blob/main/skills/productivity/grill-me/SKILL.md
```

Use `--flat` to dump a folder's contents directly into the destination instead
of creating a named subfolder. It merges into an existing directory without
deleting what's already there (add `--force` to wipe it first); with no
destination it dumps into the current directory:

```bash
degit --flat user/repo/sub ~/code/my-app   # contents land in ~/code/my-app/
degit --flat user/repo/sub                 # contents land in ./
```

If a subdirectory doesn't exist in the repo, degit fails loudly and suggests the
closest real path rather than silently producing an empty folder.

Other flags: `--force` (overwrite an existing target), `--verbose`, `--quiet`,
`--no-progress`. Clear the cache with `degit clear [filter]`.

## Installation

```bash
brew install qiushiyan/tap/degit
```
