APP_NAME := i3_next

BUILD_DIR := build

GO := go

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.commit=$(COMMIT)

-include tests/vm/vm.mk

.PHONY: all build test vet fmt clean run check

all: check build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -race ./...

check: fmt vet test

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