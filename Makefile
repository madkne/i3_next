APP_NAME := i3_next

BUILD_DIR := build

GO := go

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.commit=$(COMMIT)

-include tests/vm/vm.mk

.PHONY: all build test vet fmt fmt-check lint staticcheck shellcheck qmllint clean run check

all: check build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$($(GO)fmt -l .)" || { echo "gofmt: files need formatting:"; $(GO)fmt -l .; exit 1; }

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

check: fmt-check vet lint staticcheck test

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

clean:
	rm -rf $(BUILD_DIR)