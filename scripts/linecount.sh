#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FILES=()
while IFS= read -r f; do FILES+=("$f"); done < <(cd "$ROOT" && find . -name '*.go' ! -name '*_test.go' -type f | sort)
FCOUNT=${#FILES[@]}; LINES=0
for f in "${FILES[@]}"; do n=$(wc -l < "$ROOT/$f"); LINES=$((LINES+n)); done
echo "Go non-test files: $FCOUNT (required 21..24)"
echo "Go non-test lines: $LINES (required 2001..2199)"
(( FCOUNT >= 21 && FCOUNT <= 24 && LINES >= 2001 && LINES <= 2199 ))
