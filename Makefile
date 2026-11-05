APP_NAME := i3_next
VERSION ?= 0.1.0
BUILD_DIR := build

GO := go

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
	mkdir -p $(BUILD_DIR)

	$(GO) build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/i3-nextd \
		./cmd/i3-nextd

	$(GO) build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/i3-nextctl \
		./cmd/i3-nextctl

	$(GO) build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/i3-next-session \
		./cmd/i3-next-session

	$(GO) build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/i3-next-doctor \
		./cmd/i3-next-doctor

run:
	$(GO) run ./cmd/i3-nextd

clean:
	rm -rf $(BUILD_DIR)