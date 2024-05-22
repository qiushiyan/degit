# go-degit

Go port of [degit](https://github.com/rich-harris/degit).

```bash
degit user/repo#ref output-dir
```

Downloads the github repository locally. You can specify subdirectories and use Gitlab and Bitbucket repositories as well. If the commit hash does not change, degit uses the cached version to save downloading again.