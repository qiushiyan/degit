# Changelog

## [0.2.0](https://github.com/qiushiyan/degit/compare/v0.1.1...v0.2.0) (2026-10-05)


### Features

* cp-like destination semantics, --flat, and hard error on missing subdir ([083b28b](https://github.com/qiushiyan/degit/commit/083b28b349e7759d2a9e73d95695b650d40ac75c))


### Bug Fixes

* harden --flat edge cases and refresh stale messaging ([841e3ec](https://github.com/qiushiyan/degit/commit/841e3ec651b9eb9cca05ce1759366a2e37b08232))

## [0.1.1](https://github.com/qiushiyan/degit/compare/v0.1.0...v0.1.1) (2026-05-25)


### Bug Fixes

* **ci:** put homebrew formula in Formula/ so the tap is installable ([47ceb29](https://github.com/qiushiyan/degit/commit/47ceb290db128d80a3ffbfc622b468b366076b84))

## [0.1.0](https://github.com/qiushiyan/degit/compare/v0.0.8...v0.1.0) (2026-05-25)


### Features

* clone progress bar and feedback lines ([a33d4e3](https://github.com/qiushiyan/degit/commit/a33d4e32c7b0ad36739bf4f9152c14233e624c1a))
* **cmd:** add --no-progress and --quiet flags ([773d4b4](https://github.com/qiushiyan/degit/commit/773d4b4cfec72e4080a2a083eb5f109493858576))
* **cmd:** add CLI progress bar adapter for pkg.Progress ([7ab7516](https://github.com/qiushiyan/degit/commit/7ab7516a6fa4c9e3f73fae58fb75ee3fd9a1ca13))
* **cmd:** add status line helpers for clone feedback ([ed196c9](https://github.com/qiushiyan/degit/commit/ed196c9aadc80fd24ba3b7bc4609f8ae8835232b))
* **cmd:** wire progress bar and status lines into clone ([2e12ac5](https://github.com/qiushiyan/degit/commit/2e12ac51da7fa113013ce931c751c69511013d51))
* **pkg:** add Progress interface and wire through download ([4ac772d](https://github.com/qiushiyan/degit/commit/4ac772ddbd01b066d761cc257f904f4a6e368ae3))
* **pkg:** add Resolve method with Hash and Cached fields ([d1099e7](https://github.com/qiushiyan/degit/commit/d1099e70e5d15afd569f7999a95db1690b31ebf6))
