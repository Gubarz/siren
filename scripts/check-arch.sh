#!/usr/bin/env bash
set -euo pipefail

# Phase 1: bootstrap must never import Wails.
if go list -deps ./internal/bootstrap 2>/dev/null | grep -q 'github\.com/wailsapp/wails'; then
  echo "check-arch: bootstrap imports Wails" >&2
  go list -deps ./internal/bootstrap | grep 'github\.com/wailsapp/wails' >&2
  exit 1
fi

# Phase 2: headless binary must never import Wails.
if go list -deps ./cmd/headless 2>/dev/null | grep -q 'github\.com/wailsapp/wails'; then
  echo "check-arch: cmd/headless imports Wails" >&2
  go list -deps ./cmd/headless | grep 'github\.com/wailsapp/wails' >&2
  exit 1
fi

# Phase 3: root main must import Wails (it delegates to cmd/gui).
# This is NOT enforced — the root shim is intentionally GUI-only.

echo "check-arch: architecture rules passed"
