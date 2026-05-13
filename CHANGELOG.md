# Changelog

All notable changes to this project will be documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com); versions follow
[SemVer](https://semver.org).

## [Unreleased]

### Fixed

- Enforced notarization inside the GoReleaser build hook so releases cut via
  raw `goreleaser release --clean` are notarized (not only releases run via
  `make release`).

### Added

- `jwa-harden doctor signing` checks macOS release prerequisites:
  `codesign`, `xcrun notarytool`, `MACOS_SIGN_IDENTITY`, and the
  `notarytool` keychain profile.
- Added a dedicated CI workflow (`make check`) and a release `verify` job
  gate so publish runs only after checks pass.

## [0.1.2] — 2026-05-13

### Fixed

- **Local release uses `gh auth token` for `GITHUB_TOKEN`** instead of the
  scoped tap-writer PAT, which couldn't create releases on this repo
  (only on `homebrew-tap`). v0.1.1's release attempt failed with HTTP
  403 from the GitHub API. The tap-writer PAT remains in `.env.template`
  as `HOMEBREW_TAP_GITHUB_TOKEN` for the Cask commit step; `GITHUB_TOKEN`
  is now injected by the Makefile from your gh CLI keyring.

## [0.1.1] — 2026-05-13 (never released)

### Fixed

- **macOS Gatekeeper now passes** on first run after
  `brew install --cask jwa91/tap/jwa-harden`. v0.1.0's cask shipped
  unsigned binaries which Tahoe Gatekeeper blocks with "Apple could not
  verify jwa-harden is free of malware". This release codesigns each
  darwin binary with Developer ID + hardened runtime + secure timestamp
  (`scripts/codesign.sh` invoked as goreleaser `builds.hooks.post`) and
  submits each codesigned binary to `xcrun notarytool`
  (`scripts/notarize-darwin.sh` invoked by the Makefile release target).
  The published archive is byte-identical pre/post notarization — Apple
  records the binary's CDHash so the Gatekeeper online check passes at
  install time.
- **Linux smoke test asset name** corrected from `Linux` to `linux` so
  next CI run won't fail on case-sensitivity (carried over from the
  v0.1.0 release post-mortem fix).

### Changed

- **CI release workflow** is now `workflow_dispatch`-only until CI has
  signing credentials. Local releases via `make release VERSION=X.Y.Z`
  are canonical for now.
- **`.env.template`** gains `MACOS_SIGN_IDENTITY` (resolves the
  `make-dmg-identity` 1Password item shared with `trnscrb` and
  `jwa-tobrew`).
- **Makefile `release`** preflight now also checks for the
  `notarytool` keychain profile.

## [0.1.0] — 2026-05-13

### Added

- Initial release. `run -- <cmd>` walks up from `$PWD` to find the nearest
  `.env.template` and execs `op run --env-file=<found> -- <cmd>`.
- `doctor` subcommand reports `op` presence, signin state, and whether a
  `.env.template` is reachable from the current directory.
- `version` subcommand prints build info populated via GoReleaser ldflags.
- GoReleaser pipeline using `homebrew_casks:` (modern replacement for the
  deprecated `brews:` block per ADR 0008 in `jwa91/homebrew-tap`).
  Auto-commits `Casks/jwa-harden.rb` back into the tap on release.

[Unreleased]: https://github.com/jwa91/jwa-harden/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/jwa91/jwa-harden/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/jwa91/jwa-harden/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/jwa91/jwa-harden/releases/tag/v0.1.0
