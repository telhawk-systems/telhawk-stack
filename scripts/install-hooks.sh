#!/bin/bash
# Install Git hooks for TelHawk Stack

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"
HOOKS_SOURCE_DIR="$SCRIPT_DIR/hooks"

echo "Installing Git hooks for TelHawk Stack..."
echo ""

# Check if we're in a git repository
if [ ! -d "$REPO_ROOT/.git" ]; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Install pre-commit hook
if [ -f "$HOOKS_SOURCE_DIR/pre-commit" ]; then
    echo "Installing pre-commit hook (auto-format Go code with gofmt)..."
    cp "$HOOKS_SOURCE_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
    chmod +x "$HOOKS_DIR/pre-commit"
    echo "✓ pre-commit hook installed"
else
    echo "Warning: pre-commit hook not found in $HOOKS_SOURCE_DIR"
fi

echo ""
echo "Git hooks installed successfully!"
echo ""
echo "Installed hooks:"
echo "  - pre-commit: Automatically format Go code with gofmt"
echo ""
echo "To disable hooks temporarily, use: git commit --no-verify"
