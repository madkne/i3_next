APP_NAME := i3_next

PREFIX ?= /usr/local
DESTDIR ?=

BUILD_DIR := build

GO := go

GOFMT := gofmt

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w \
	-X i3_next/internal/version.Commit=$(COMMIT) \
	-X i3_next/internal/version.BuildDate=$(BUILD_DATE)

.DEFAULT_GOAL := all

-include tests/vm/vm.mk

.PHONY: all build test vet fmt fmt-check lint staticcheck shellcheck qmllint naming check-naming clean run check

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
	staticcheck ./...

shellcheck:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not installed; skipping"; exit 0; }
	@find . -type f -name '*.sh' -print0 | xargs -0 -r shellcheck

qmllint:
	@command -v qmllint >/dev/null 2>&1 || { echo "qmllint not installed; skipping"; exit 0; }
	@find ui -type f -name '*.qml' -print0 | xargs -0 -r qmllint

test:
	$(GO) test -race ./...

naming:
	bash scripts/check-naming.sh

check-naming: naming

check: fmt-check vet lint staticcheck test naming

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
	@install -m 0644 configs/sway/config configs/sway/outputs configs/sway/keybindings configs/sway/window-rules "$(DESTDIR)$(PREFIX)/share/i3-next/sway/"
	@install -d "$(DESTDIR)$(PREFIX)/share/i3-next/defaults"
	@install -m 0644 configs/defaults/* "$(DESTDIR)$(PREFIX)/share/i3-next/defaults/"
	@install -d "$(DESTDIR)$(PREFIX)/share/i3-next/quickshell"
	@cp -a ui/quickshell/. "$(DESTDIR)$(PREFIX)/share/i3-next/quickshell/"
	@install -d "$(DESTDIR)$(PREFIX)/share/wayland-sessions"
	@install -m 0644 packaging/wayland-sessions/i3-next.desktop "$(DESTDIR)$(PREFIX)/share/wayland-sessions/"
	@install -d "$(DESTDIR)$(PREFIX)/share/doc/i3-next"
	@cp -a docs/. "$(DESTDIR)$(PREFIX)/share/doc/i3-next/"
	@rm -f "$(DESTDIR)$(PREFIX)/share/doc/i3-next/tmp.md"
	@install -m 0644 README.md SECURITY.md CONTRIBUTING.md LICENSE "$(DESTDIR)$(PREFIX)/share/doc/i3-next/" 2>/dev/null || true
	@echo "Installed i3_next to $(DESTDIR)$(PREFIX)"