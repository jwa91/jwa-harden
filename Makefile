.PHONY: build install dev release check clean

BIN_DIR ?= $(HOME)/.local/bin

build:
	go build -o bin/jwa-harden ./cmd/jwa-harden

install: build
	install -d $(BIN_DIR)
	install -m 0755 bin/jwa-harden $(BIN_DIR)/jwa-harden
	@echo "installed jwa-harden to $(BIN_DIR)/jwa-harden"
	@echo "ensure $(BIN_DIR) is on your PATH"

dev: install

check:
	go vet ./...
	go test ./...

# Local release. CI is disabled (workflow_dispatch only) until signing
# creds are in CI; until then every release runs locally. Requires
# 1Password signed in, the v$(VERSION) tag at HEAD, and a keychain
# profile named "notarytool" (xcrun notarytool store-credentials).
#
# Bootstrap note: this Makefile cannot wrap with `jwa-harden run`
# because jwa-harden may be the thing we're releasing. Falls back to
# `op run --env-file=.env.template --` directly.
release:
	@test -n "$(VERSION)" || (echo "usage: make release VERSION=X.Y.Z" && exit 2)
	@op whoami >/dev/null || (echo "1Password not signed in: eval \$$(op signin)" && exit 1)
	@gh auth status >/dev/null 2>&1 || (echo "gh not authenticated: gh auth login" && exit 1)
	@xcrun notarytool history --keychain-profile notarytool >/dev/null 2>&1 || \
	  (echo "keychain profile 'notarytool' missing — see scripts/notarize-darwin.sh header"; exit 1)
	@existing=$$(git rev-parse -q --verify "v$(VERSION)^{commit}" 2>/dev/null); \
	head=$$(git rev-parse HEAD); \
	test -n "$$existing" && test "$$existing" = "$$head" || \
	  (echo "v$(VERSION) must exist and point at HEAD before release"; exit 3)
	# Build + codesign + archive + publish + commit Cask back to the tap.
	# GITHUB_TOKEN is the user's gh CLI token (repo scope, can create
	# releases on this repo). HOMEBREW_TAP_GITHUB_TOKEN (op-resolved) is
	# the fine-grained tap-writer PAT used for the Cask commit.
	# Codesign happens inside goreleaser's builds.hooks.post.
	GITHUB_TOKEN="$$(gh auth token)" \
	  op run --env-file=.env.template -- goreleaser release --clean
	# Submit each codesigned darwin binary to notarytool. The published
	# archive is byte-identical pre/post — Apple records the binary's
	# CDHash so Gatekeeper online-check passes on first install.
	scripts/notarize-darwin.sh jwa-harden $(VERSION)

clean:
	rm -rf bin dist
