# Changelog

All notable changes to this project will be documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com); versions follow
[SemVer](https://semver.org).

## [Unreleased]

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

[Unreleased]: https://github.com/jwa91/jwa-harden/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/jwa91/jwa-harden/releases/tag/v0.1.0
