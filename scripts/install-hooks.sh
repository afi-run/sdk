#!/usr/bin/env bash
#
# Installs the AFI SDK git hooks by pointing `core.hooksPath` at scripts/git-hooks.
# Run once after cloning the repository.
#
#   bash scripts/install-hooks.sh
#

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

HOOKS_DIR="scripts/git-hooks"

if [[ ! -d "$HOOKS_DIR" ]]; then
  echo "$HOOKS_DIR not found — are you running from the repo root?" >&2
  exit 1
fi

chmod +x "$HOOKS_DIR"/*

current="$(git config --get core.hooksPath 2>/dev/null || echo "")"
if [[ "$current" == "$HOOKS_DIR" ]]; then
  echo "core.hooksPath already set to $HOOKS_DIR — nothing to do."
  exit 0
fi

git config core.hooksPath "$HOOKS_DIR"
echo "Set core.hooksPath = $HOOKS_DIR"
echo "Hooks installed:"
ls -1 "$HOOKS_DIR" | sed 's/^/  /'
echo
echo "Bypass once with:  SKIP_PRE_PUSH=1 git push"
echo "Force every step:  PRE_PUSH_ALL=1 git push"
echo "Disable entirely:  git config --unset core.hooksPath"
