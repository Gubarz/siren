#!/usr/bin/env bash
set -euo pipefail

# Phase 1: automation package must have no project-internal (other than itself),
# Wails, or Sliver imports.
if go list -deps ./internal/automation 2>/dev/null | grep -v '^sliver-gui/internal/automation$' | grep -qE \
    'github\.com/wailsapp/wails|github\.com/bishopfox/sliver|sliver-gui/internal/'; then
  echo "check-arch: automation contains forbidden dependencies" >&2
  go list -deps ./internal/automation | grep -v '^sliver-gui/internal/automation$' | grep -E \
    'github\.com/wailsapp/wails|github\.com/bishopfox/sliver|sliver-gui/internal/' >&2
  exit 1
fi

# Phase 2: bootstrap must never import Wails.
if go list -deps ./internal/bootstrap 2>/dev/null | grep -q 'github\.com/wailsapp/wails'; then
  echo "check-arch: bootstrap imports Wails" >&2
  go list -deps ./internal/bootstrap | grep 'github\.com/wailsapp/wails' >&2
  exit 1
fi

echo "check-arch: architecture rules passed"
