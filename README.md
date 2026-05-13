# jwa-harden

Wrap a command with `op run` against the nearest `.env.template`.

Part of the `jwa-*` family of personal CLIs published via
[`jwa91/homebrew-tap`](https://github.com/jwa91/homebrew-tap).

```bash
brew install jwa91/tap/jwa-harden
```

## What it does

`jwa-harden run -- <cmd>` walks up from `$PWD` to find the nearest
`.env.template`, then runs:

```
op run --env-file=<found> -- <cmd>
```

That's the entire scope. Secret resolution is delegated to 1Password CLI;
this binary owns only the discovery + exec.

## Usage

```bash
# Inside a repo with a .env.template (homebrew-tap, a jwa-tobrew project, etc.)
jwa-harden run -- goreleaser release --clean
jwa-harden run -- gh release create v1.2.3

# Check tooling
jwa-harden doctor

# Print version
jwa-harden version
```

## Why this exists

Every `jwa-*` release flow needs `$GITHUB_TOKEN` resolved from 1Password,
and every project's `.env.template` lives at the repo root. Hand-typing
`op run --env-file=.env.template --` everywhere is friction; passing a
path through `make` is brittle. This binary closes that gap with
auto-discovery so the same one-liner works from any subdirectory.

See [ADR 0008](https://github.com/jwa91/homebrew-tap/blob/main/docs/adr/0008-one-binary-per-repo-and-homebrew-casks.md)
in the tap repo for the structural decision (one binary per repo;
`homebrew_casks` over the deprecated `brews:` block).

## Security model

`jwa-harden` reads no secrets directly. It only:

1. Stats `.env.template` files (file existence + non-directory check).
2. Execs `op` with arguments derived from the discovered path.

`op` does all the resolution; the resolved secrets exist only in the
spawned child process's environment, never on disk. See
[`~/dotfiles/docs/security-ground-rules.md`](https://github.com/jwa91/dotfiles)
for the full model.

## License

MIT
