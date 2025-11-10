#!/usr/bin/env bash
set -euo pipefail

# Reports outdated npm packages for each package.json directory.
# Usage: ./scripts/check-npm-updates.sh | tee ./tmp/npm-updates.txt

if ! command -v rg >/dev/null 2>&1; then
  echo "ripgrep (rg) is required. Please install rg and retry." >&2
  exit 1
fi

projects=$(rg -l --glob '!**/node_modules/**' '^\{' --no-messages **/package.json | sed 's|/package.json||' | sort)
if [ -z "${projects}" ]; then
  echo "No package.json files found." >&2
  exit 0
fi

for p in ${projects}; do
  echo "===> ${p}"
  (
    cd "${p}"
    if command -v jq >/dev/null 2>&1; then
      npm outdated --json 2>/dev/null \
        | jq -r 'to_entries[]? | "\(.key) current=\(.value.current) wanted=\(.value.wanted) latest=\(.value.latest) type=\(.value.type // \"prod\")"' \
        || true
    else
      npm outdated || true
    fi
  )
  echo
done

