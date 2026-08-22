#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

violations=$(
  find ./cmd ./internal -name '*.go' -not -path '*/node_modules/*' -print0 |
    xargs -0 grep -nE '^[[:space:]]*(//|/\*)' |
    grep -vE '//(go:(embed|build|generate)|line )' |
    grep -vE '//nolint' |
    grep -vE '// Code generated' ||
    true
)

if [ -n "$violations" ]; then
  echo "❌ This codebase carries no comments — only //go: directives, //nolint and 'Code generated' are allowed."
  echo "   Put the reasoning in CLAUDE.md or .claude/skills/architecture (Invariant 7) instead."
  echo
  echo "$violations"
  exit 1
fi

echo "✅ no comments found"
