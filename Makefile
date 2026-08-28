.PHONY: build test test-integration lint fmt vet clean release-snapshot install help

VERSION ?= $(shell git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/drackthor/mdformat/internal/version.BuildVersion=$(VERSION)

## build: Build the binary for the current platform
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o mdformat .

## install: Install the binary into GOBIN
install:
	go install -trimpath -ldflags="$(LDFLAGS)" .

## test: Run all tests with race detection
test:
	go test -race ./...

## test-integration: Run the golden-file integration tests in test-cases/
test-integration:
	go test -race ./test-cases/...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go source files
fmt:
	golangci-lint fmt

## vet: Run go vet
vet:
	go vet ./...

## release-snapshot: Build multi-arch binaries locally via goreleaser (no publish)
release-snapshot:
	goreleaser release --snapshot --clean

## clean: Remove build artifacts
clean:
	rm -rf mdformat dist/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
