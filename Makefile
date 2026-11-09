APP_NAME := i3_next

PREFIX ?= /usr/local
DESTDIR ?=
WALLPAPER_DIR ?= /usr/share/i3-next/wallpapers

BUILD_DIR := build

GO := go

# Single source of truth for Go tool versions (dev-setup.sh, pre-commit and CI pin these).
STATICCHECK_VERSION := v0.8.2
GOLANGCI_LINT_VERSION := v2.15.0
GOVULNCHECK_VERSION := v1.1.4

export PATH := $(HOME)/go/bin:$(PATH)

GOFMT := gofmt

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w \
	-X i3_next/internal/version.Commit=$(COMMIT) \
	-X i3_next/internal/version.BuildDate=$(BUILD_DATE)

.DEFAULT_GOAL := all

-include tests/vm/vm.mk

.PHONY: all build test vet fmt fmt-check lint staticcheck shellcheck qmllint naming check-naming clean run check install push install-lint-tools

all: check build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$($(GOFMT) -l .)" || { echo "gofmt: files need formatting:"; $(GOFMT) -l .; exit 1; }

lint:
	golangci-lint run ./...

staticcheck:
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; else echo "staticcheck not installed; skipping"; fi

install-lint-tools:
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

shellcheck:
	@if command -v shellcheck >/dev/null 2>&1; then find . -type f -name '*.sh' -print0 | xargs -0 -r shellcheck; else echo "shellcheck not installed; skipping"; fi

qmllint:
	@if command -v qmllint >/dev/null 2>&1; then find ui -type f -name '*.qml' -print0 | xargs -0 -r qmllint; else echo "qmllint not installed; skipping"; fi

test:
	$(GO) test -race ./...

naming:
	bash scripts/check-naming.sh

check-naming: naming

check: fmt-check vet lint staticcheck test naming
push:
	@bash scripts/git-push.sh $(PUSH_ARGS)

build:
	@mkdir -p $(BUILD_DIR)
	@echo "-------------------------------"
	@echo "Building i3-nextd..."
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/i3-nextd ./cmd/i3-nextd
	@echo "-------------------------------"
	@echo "Building i3-nextctl..."
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/i3-nextctl ./cmd/i3-nextctl
	@echo "-------------------------------"
	@echo "Building i3-next-session..."
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/i3-next-session ./cmd/i3-next-session
	@echo "-------------------------------"
	@echo "Building i3-next-doctor..."
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/i3-next-doctor ./cmd/i3-next-doctor

run:
	$(GO) run ./cmd/i3-nextd

install: build
	@install -d "$(DESTDIR)$(PREFIX)/bin"
	@install -m 0755 build/i3-nextd build/i3-nextctl build/i3-next-session build/i3-next-doctor "$(DESTDIR)$(PREFIX)/bin/"
	@install -d "$(DESTDIR)$(PREFIX)/share/i3-next/sway"
	@mkdir -p "$(DESTDIR)$(WALLPAPER_DIR)"
	@install -Dm644 assets/wallpapers/i3-next-default.png $(DESTDIR)$(WALLPAPER_DIR)/i3-next-default.png
	@install -m 0644 configs/sway/config configs/sway/outputs configs/sway/keybindings configs/sway/window-rules "$(DESTDIR)$(PREFIX)/share/i3-next/sway/"
	@install -d "$(DESTDIR)$(PREFIX)/share/i3-next/defaults"
	@install -m 0644 internal/core/config/defaults/* "$(DESTDIR)$(PREFIX)/share/i3-next/defaults/"
	@install -d "$(DESTDIR)$(PREFIX)/share/i3-next/quickshell"
	@cp -a ui/quickshell/. "$(DESTDIR)$(PREFIX)/share/i3-next/quickshell/"
	@install -d "$(DESTDIR)$(PREFIX)/share/wayland-sessions"
	@install -m 0644 packaging/wayland-sessions/i3-next.desktop "$(DESTDIR)$(PREFIX)/share/wayland-sessions/"
	@install -d "$(DESTDIR)$(PREFIX)/share/doc/i3-next"
	@cp -a docs/. "$(DESTDIR)$(PREFIX)/share/doc/i3-next/"
	@rm -f "$(DESTDIR)$(PREFIX)/share/doc/i3-next/tmp.md"
	@install -m 0644 README.md SECURITY.md CONTRIBUTING.md LICENSE "$(DESTDIR)$(PREFIX)/share/doc/i3-next/" 2>/dev/null || true
	@echo "Installed i3_next to $(DESTDIR)$(PREFIX)"

