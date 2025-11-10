#!/usr/bin/env bash
set -euo pipefail

# Report available updates for Go modules in each directory with a go.mod.
# Usage: ./scripts/check-go-updates.sh
# Pipe to a file if desired: ./scripts/check-go-updates.sh | tee ./tmp/output.txt

if ! command -v rg >/dev/null 2>&1; then
  echo "ripgrep (rg) is required. Please install rg and retry." >&2
  exit 1
fi

mods=$(rg -l --glob '!**/vendor/**' --glob '!**/node_modules/**' '^module ' --no-messages **/go.mod | sed 's|/go.mod||' | sort)

if [ -z "${mods}" ]; then
  echo "No go.mod files found." >&2
  exit 0
fi

for m in ${mods}; do
  echo "===> ${m}"
  (
    cd "${m}"
    # List outdated modules (direct and transitive). Filter to those with updates.
    # Prefer jq if available for robust JSON parsing; fallback to text format otherwise.
    if command -v jq >/dev/null 2>&1; then
      go list -m -u -json all 2>/dev/null \
        | jq -r 'select(.Path != null and .Main != true and .Update != null) | "\(.Path) \(.Version) -> \(.Update.Version)"' \
        | sort || true
    else
      go list -m -u -f '{{if and (not .Main) .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all \
        | sort || true
    fi
  )
  echo
done

