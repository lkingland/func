#!/usr/bin/env bash
#
# Emit project files for code-quality checks (lint, goimports, misspell,
# whitespace, EOF), excluding generated/vendored/external paths.
#
# Detects VCS:
#   git  - `git ls-files` + .gitattributes filtering (linguist-generated,
#          linguist-vendored). This is the canonical CI behavior.
#   jj   - `jj file list` when no .git directory is present (e.g. jj secondary
#          workspaces). .gitattributes filtering is not applied — jj has no
#          equivalent built-in — so path-based excludes catch the common
#          generated/vendored cases.
#
# Used by the check-* targets in the top-level Makefile.

set -euo pipefail

if git rev-parse --git-dir >/dev/null 2>&1; then
    git ls-files \
      | git check-attr --stdin linguist-generated \
      | grep -Ev ': (set|true)$' \
      | cut -d: -f1 \
      | git check-attr --stdin linguist-vendored \
      | grep -Ev ': (set|true)$' \
      | cut -d: -f1
elif [ -d .jj ] && command -v jj >/dev/null 2>&1; then
    jj file list
else
    echo "list-files.sh: no git or jj repository found" >&2
    exit 1
fi | grep -Ev '(vendor/|third_party/|\.git)'
