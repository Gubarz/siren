.PHONY: dev dev-backend analyze analyze-go analyze-frontend build print-version print-ldflags

GOCACHE ?= /tmp/sliver-gui-cache/go-build
XDG_CACHE_HOME ?= /tmp/sliver-gui-cache
CCACHE_DIR ?= /tmp/sliver-gui-cache/ccache
BUILD_OUTPUT ?= /tmp/sliver-gui-build
BUILD_TAGS ?=
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILDINFO_LDFLAGS := -X sliver-gui/internal/buildinfo.Version=$(VERSION) -X sliver-gui/internal/buildinfo.Commit=$(COMMIT) -X sliver-gui/internal/buildinfo.Date=$(BUILD_DATE)
BUILD_TAGS_ARG := $(if $(strip $(BUILD_TAGS)),-tags "$(BUILD_TAGS)",)

export GOCACHE
export XDG_CACHE_HOME
export CCACHE_DIR

dev:
	wails dev -nogorebuild $(BUILD_TAGS_ARG) -ldflags "$(BUILDINFO_LDFLAGS)"

dev-backend:
	wails dev $(BUILD_TAGS_ARG) -ldflags "$(BUILDINFO_LDFLAGS)"

analyze: analyze-go analyze-frontend

analyze-go:
	go test ./...
	go vet .
	staticcheck .
	deadcode -test .
	dupl -t 50 *.go
	# Function length (45) via golangci-lint funlen; file length (350) via
	# our shell check. Both budgets documented in CONTRIBUTING.md.
	golangci-lint run --timeout=5m ./...
	./scripts/check-go-file-length.sh
	./scripts/check-arch.sh
	./scripts/check-import-boundaries.sh

analyze-frontend:
	npm --prefix frontend run analyze

build:
	npm --prefix frontend run build
	go build $(BUILD_TAGS_ARG) -ldflags "$(BUILDINFO_LDFLAGS)" -o $(BUILD_OUTPUT) .

print-version:
	@printf '%s\n' "$(VERSION)"

print-ldflags:
	@printf '%s\n' "$(BUILDINFO_LDFLAGS)"
