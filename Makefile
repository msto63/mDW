.PHONY: all build build-all run run-all test lint clean proto docker-build docker-run dev help \
	test-integration test-integration-quick test-integration-turing test-integration-russell

# Variables
SERVICE ?= kant
BINARY_NAME = mdw
GO_FILES = $(shell find . -name '*.go' -not -path './foundation/*')
PROTO_DIR = api/proto
PROTO_OUT = api/gen

# Version Management - Read from VERSION file and auto-increment patch on build
VERSION_FILE = VERSION
CURRENT_VERSION := $(shell cat $(VERSION_FILE) 2>/dev/null || echo "0.0.0")
VERSION_PARTS := $(subst ., ,$(CURRENT_VERSION))
VERSION_MAJOR := $(word 1,$(VERSION_PARTS))
VERSION_MINOR := $(word 2,$(VERSION_PARTS))
VERSION_PATCH := $(word 3,$(VERSION_PARTS))
NEXT_PATCH := $(shell echo $$(($(VERSION_PATCH) + 1)))
NEXT_VERSION := $(VERSION_MAJOR).$(VERSION_MINOR).$(NEXT_PATCH)

# Build info
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/msto63/mDW/cmd/mdw/cmd.Version=$(NEXT_VERSION) \
	-X github.com/msto63/mDW/cmd/mdw/cmd.GitCommit=$(GIT_COMMIT) \
	-X github.com/msto63/mDW/cmd/mdw/cmd.BuildDate=$(BUILD_DATE) \
	-X github.com/msto63/mDW/internal/tui/chatclient.Version=$(NEXT_VERSION) \
	-X github.com/msto63/mDW/internal/tui/chatclient.BuildTime=$(BUILD_DATE) \
	-X github.com/msto63/mDW/internal/tui/chatclient.GitCommit=$(GIT_COMMIT)"

# Default target
all: build

# ─────────────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────────────

## build: Build the CLI binary (auto-increments version)
build:
	@echo "Building $(BINARY_NAME) v$(NEXT_VERSION)..."
	@echo "$(NEXT_VERSION)" > $(VERSION_FILE)
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/mdw
	@echo "Build complete: v$(NEXT_VERSION)"

## build-linux: Build for Linux (for containers)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux ./cmd/mdw

## build-all: Build all service binaries
build-all: build

# ─────────────────────────────────────────────────────────────────
# Run
# ─────────────────────────────────────────────────────────────────

## run: Run a specific service (SERVICE=kant)
run: build
	@./bin/$(BINARY_NAME) serve $(SERVICE)

## run-all: Run all services
run-all: build
	@./bin/$(BINARY_NAME) serve

## dev: Run with hot reload (requires air)
dev:
	@which air > /dev/null || go install github.com/air-verse/air@latest
	@air -c .air.toml

# ─────────────────────────────────────────────────────────────────
# Test & Lint
# ─────────────────────────────────────────────────────────────────

## test: Run all tests
test:
	@go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-integration: Run integration tests (requires running services)
test-integration:
	@echo "Running integration tests..."
	@echo "Note: Services must be running (make run-all)"
	@go test -v -tags=integration ./test/integration/...

## test-integration-quick: Run quick health check integration tests
test-integration-quick:
	@echo "Running quick health check tests..."
	@go test -v -tags=integration -run TestQuickHealthChecks ./test/integration/grpc/...

## test-integration-turing: Run Turing integration tests only
test-integration-turing:
	@echo "Running Turing integration tests..."
	@go test -v -tags=integration -run TestTuring ./test/integration/grpc/...

## test-integration-russell: Run Russell integration tests only
test-integration-russell:
	@echo "Running Russell integration tests..."
	@go test -v -tags=integration -run TestRussell ./test/integration/grpc/...

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run

## fmt: Format code
fmt:
	@go fmt ./...
	@gofmt -s -w .

## vet: Run go vet
vet:
	@go vet ./...

# ─────────────────────────────────────────────────────────────────
# Proto
# ─────────────────────────────────────────────────────────────────

## proto: Generate Go code from proto files
proto:
	@echo "Generating protobuf code..."
	@rm -rf $(PROTO_OUT)
	@mkdir -p $(PROTO_OUT)
	@protoc --go_out=. --go_opt=module=github.com/msto63/mDW \
		--go-grpc_out=. --go-grpc_opt=module=github.com/msto63/mDW \
		-I$(PROTO_DIR) \
		$(PROTO_DIR)/*.proto
	@echo "Proto generation complete"

## proto-install: Install protoc plugins
proto-install:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# ─────────────────────────────────────────────────────────────────
# Container (Podman/Docker)
# ─────────────────────────────────────────────────────────────────

## podman-build: Build all container images
podman-build:
	@echo "Building container images..."
	@podman-compose build

## podman-up: Start all services in containers
podman-up:
	@podman-compose up -d

## podman-down: Stop all services
podman-down:
	@podman-compose down

## podman-logs: Show logs for all services
podman-logs:
	@podman-compose logs -f

## podman-ps: Show running containers
podman-ps:
	@podman-compose ps

# ─────────────────────────────────────────────────────────────────
# Utility
# ─────────────────────────────────────────────────────────────────

## clean: Remove build artifacts
clean:
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -rf $(PROTO_OUT)

## deps: Download dependencies
deps:
	@go mod download
	@go mod tidy

## version: Show version info
version:
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

## status: Show service status
status: build
	@./bin/$(BINARY_NAME) status

# ─────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────

## help: Show this help message
help:
	@echo "meinDENKWERK - Makefile Commands"
	@echo "════════════════════════════════════════════════════════════"
	@echo ""
	@echo "Build:"
	@grep -E '^## build' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Run:"
	@grep -E '^## (run|dev)' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Test & Lint:"
	@grep -E '^## (test|lint|fmt|vet)' Makefile | sed 's/## /  /' | head -20
	@echo ""
	@echo "Proto:"
	@grep -E '^## proto' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Container:"
	@grep -E '^## podman' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Utility:"
	@grep -E '^## (clean|deps|version|status|help)' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Usage: make [target] [SERVICE=name]"
	@echo "Example: make run SERVICE=turing"
