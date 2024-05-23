# go-degit

Go port of [degit](https://github.com/rich-harris/degit).

Usage

```bash
degit user/repo#ref output-dir
```

This downloads the github repository `https://github.com/user/repo" at "ref", ref could be a branch, a tag or commit hash. If ref is empty, the main branch will be used. You can specify subdirectories and use Gitlab and Bitbucket repositories as well. degit also maintains a cache to save downloads and keep refs updated.

## Installation

```bash
brew tap qiushiyan/degit https://github.com/qiushiyan/degit
brew install qiushiyan/degit/degit

```
