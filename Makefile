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

# Local release. Requires 1Password signed in. Tag v$(VERSION) must
# already exist on HEAD; create with `git tag -a v$(VERSION) -m "..."`
# and push it before running this target. CI handles the same flow
# automatically for tag pushes.
release:
	@test -n "$(VERSION)" || (echo "usage: make release VERSION=0.1.0" && exit 2)
	@op whoami >/dev/null || (echo "1Password not signed in: eval \$$(op signin)" && exit 1)
	@existing=$$(git rev-parse -q --verify "v$(VERSION)^{commit}" 2>/dev/null); \
	head=$$(git rev-parse HEAD); \
	test -n "$$existing" && test "$$existing" = "$$head" || \
	  (echo "v$(VERSION) must exist and point at HEAD before release"; exit 3)
	op run --env-file=.env.template -- goreleaser release --clean

clean:
	rm -rf bin dist
