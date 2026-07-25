#!/usr/bin/env bash
# Enforce the layer decoupling contract (docs/superpowers/specs/
# 2026-07-24-substrate-bus-journal-design.md). A package "violates" the
# contract when its transitive dependency set contains a forbidden import.
set -euo pipefail

fail=0

# external_imports lists transitive deps whose first path segment contains a
# dot (i.e. non-stdlib).
external_imports() {
    go list -deps "$1" 2>/dev/null | grep -E '^[^/]*\.[^/]*/' || true
}

internal_imports() {
    go list -deps "$1" 2>/dev/null | grep '^sliver-gui/internal/' || true
}

# Rule 1: internal/bus is stdlib-only.
if out=$(external_imports ./internal/bus) && [ -n "$out" ]; then
    echo "boundary: internal/bus must import stdlib only:" >&2
    echo "$out" >&2
    fail=1
fi

# Rule 2: internal/journal imports no sliver/wails and only internal/bus internally.
# (google/uuid and other non-sliver externals are allowed.)
if out=$(external_imports ./internal/journal | grep -E '^github\.com/(bishopfox|wailsapp)') && [ -n "$out" ]; then
    echo "boundary: internal/journal must not import sliver or wails:" >&2
    echo "$out" >&2
    fail=1
fi
if out=$(internal_imports ./internal/journal | grep -v '^sliver-gui/internal/\(journal\|bus\)$') && [ -n "$out" ]; then
    echo "boundary: internal/journal may only import internal/bus:" >&2
    echo "$out" >&2
    fail=1
fi

# Rule 3: internal/localstate/journal imports no sliver/wails and only internal/journal internally.
if out=$(external_imports ./internal/localstate/journal | grep -E '^github\.com/(bishopfox|wailsapp)') && [ -n "$out" ]; then
    echo "boundary: internal/localstate/journal must not import sliver or wails:" >&2
    echo "$out" >&2
    fail=1
fi
if out=$(internal_imports ./internal/localstate/journal | grep -v '^sliver-gui/internal/\(localstate/journal\|journal\|bus\)$') && [ -n "$out" ]; then
    echo "boundary: internal/localstate/journal may only import internal/journal:" >&2
    echo "$out" >&2
    fail=1
fi

# Rule 4: internal/automation imports no sliver/wails and only bus+journal internally.
if out=$(external_imports ./internal/automation | grep -E '^github\.com/(bishopfox|wailsapp)') && [ -n "$out" ]; then
    echo "boundary: internal/automation must not import sliver or wails:" >&2
    echo "$out" >&2
    fail=1
fi
if out=$(internal_imports ./internal/automation | grep -v '^sliver-gui/internal/\(automation\|bus\|journal\)$') && [ -n "$out" ]; then
    echo "boundary: internal/automation may only import internal/bus and internal/journal:" >&2
    echo "$out" >&2
    fail=1
fi

# Rule 5: within internal/sliver/*, only automationexec may import internal/automation.
while IFS= read -r pkg; do
    case "$pkg" in
        sliver-gui/internal/sliver/automationexec) continue ;;
    esac
    if go list -f '{{join .Imports " "}}' "$pkg" | tr ' ' '\n' | grep -q '^sliver-gui/internal/automation$'; then
        echo "boundary: $pkg imports internal/automation (only automationexec may)" >&2
        fail=1
    fi
done < <(go list ./internal/sliver/...)

if [ "$fail" -ne 0 ]; then
    echo "check-import-boundaries: FAILED" >&2
    exit 1
fi
echo "check-import-boundaries: all boundary rules passed"
