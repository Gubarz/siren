#!/usr/bin/env bash
# Enforce the 350-line .go file budget. golangci-lint has no built-in
# file-length linter, so this shell check plugs the gap. See CONTRIBUTING.md.
#
# app.go is the sanctioned exception — it fans out to every App method
# and grew organically as the App-method surface.

set -euo pipefail

MAX_LINES=350
EXEMPT=("./app.go")

fail=0

while IFS= read -r -d '' file; do
    for skip in "${EXEMPT[@]}"; do
        [[ "$file" == "$skip" ]] && continue 2
    done
    lines=$(wc -l < "$file")
    if (( lines > MAX_LINES )); then
        printf '%s: %d lines (> %d)\n' "$file" "$lines" "$MAX_LINES" >&2
        fail=1
    fi
done < <(
    find . \
        -type d \( -name node_modules -o -name frontend -o -name build -o -name dist -o -name .git \) -prune \
        -o -type f -name '*.go' -not -name '*_test.go' -print0
)

if (( fail )); then
    echo "" >&2
    echo "Go file-length budget exceeded. Budgets are documented in CONTRIBUTING.md." >&2
    exit 1
fi
