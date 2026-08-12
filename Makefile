.PHONY: dev dev-backend analyze analyze-go analyze-frontend build print-version print-ldflags

GOCACHE ?= /tmp/siren-cache/go-build
XDG_CACHE_HOME ?= /tmp/siren-cache
CCACHE_DIR ?= /tmp/siren-cache/ccache
BUILD_OUTPUT ?= /tmp/siren-build
BUILD_TAGS ?=
WAILS_PORT ?= 9245
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILDINFO_LDFLAGS := -X siren/internal/buildinfo.Version=$(VERSION) -X siren/internal/buildinfo.Commit=$(COMMIT) -X siren/internal/buildinfo.Date=$(BUILD_DATE)
BUILD_OUTPUT_NAME := $(notdir $(BUILD_OUTPUT))

# Production builds need the `production` tag (wails v3); BUILD_TAGS carries
# any extras (e.g. BUILD_TAGS=gtk3 for webkit2gtk-4.1 systems).
PRODUCTION_TAGS := $(strip production $(BUILD_TAGS))
# Windows GUI apps must not allocate a console.
HOST_GOOS := $(shell go env GOOS)
GUI_LDFLAGS := $(if $(filter windows,$(or $(GOOS),$(HOST_GOOS))),-H windowsgui,)

export GOCACHE
export XDG_CACHE_HOME
export CCACHE_DIR
export BUILD_TAGS

dev:
	wails3 dev -port $(WAILS_PORT)

# v3 has no separate frontend-only mode: the vite dev server hot-reloads the
# UI while wails3 dev rebuilds the Go binary on change.
dev-backend: dev

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
	wails3 generate bindings
	npm --prefix frontend run build
	go build -tags "$(PRODUCTION_TAGS)" -trimpath -buildvcs=false -ldflags="-w -s $(GUI_LDFLAGS) $(BUILDINFO_LDFLAGS)" -o "build/bin/$(BUILD_OUTPUT_NAME)" .
	mkdir -p "$(dir $(BUILD_OUTPUT))"
	cp "build/bin/$(BUILD_OUTPUT_NAME)" "$(BUILD_OUTPUT)"

print-version:
	@printf '%s\n' "$(VERSION)"

print-ldflags:
	@printf '%s\n' "$(BUILDINFO_LDFLAGS)"
