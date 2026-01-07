# turtlectl Makefile

BINARY_NAME := turtlectl
BUILD_DIR := build
INSTALL_DIR := /usr/bin

# Version info from git
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags (matching PKGBUILD)
GOFLAGS := -buildmode=pie -trimpath
LDFLAGS := -linkmode=external -X github.com/bnema/turtlectl/cmd.version=$(VERSION) -X github.com/bnema/turtlectl/cmd.commit=$(COMMIT)

.PHONY: all build install uninstall clean fmt vet test tidy run help registry-gen update-registry

all: fmt build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .

install:
	@echo "Run the following command to install:"
	@echo "  sudo install -Dm755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)"

uninstall:
	@echo "Run the following command to uninstall:"
	@echo "  sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)"

clean:
	rm -rf $(BUILD_DIR)

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

run:
	go run .

# Registry generator (for CI and manual updates)
registry-gen:
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/registry-gen ./cmd/registry-gen

update-registry: registry-gen
	@if [ -z "$$GITHUB_TOKEN" ] && [ -z "$$GH_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN or GH_TOKEN must be set"; \
		exit 1; \
	fi
	$(BUILD_DIR)/registry-gen --output data/addons.json

help:
	@echo "turtlectl Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all       (default) Format and build"
	@echo "  build     Build binary to $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  install   Show install command (requires sudo)"
	@echo "  uninstall Show uninstall command (requires sudo)"
	@echo "  clean     Remove $(BUILD_DIR)/ directory"
	@echo "  fmt       Run go fmt ./..."
	@echo "  vet       Run go vet ./..."
	@echo "  test      Run go test ./..."
	@echo "  tidy      Run go mod tidy"
	@echo "  run       Run with go run ."
	@echo "  registry-gen    Build the registry generator tool"
	@echo "  update-registry Generate data/addons.json (requires GITHUB_TOKEN)"
	@echo "  help      Show this help"
	@echo ""
	@echo "Version: $(VERSION) ($(COMMIT))"
