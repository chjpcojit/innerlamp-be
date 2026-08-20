#!/usr/bin/env bash
#
# post_tool_use.sh
# PostToolUse hook for Claude Code CLI — Go projects.
# Runs after the agent edits/writes a file. Order: fail fast on cheap checks first.
#
# Exit code contract (Claude Code convention):
#   0 -> all checks passed
#   2 -> checks failed, stderr is fed back to the agent for self-correction
#
# Expected input: Claude Code passes hook context as JSON on stdin.
# We only need the list of changed file paths from it.
echo "$INPUT_JSON" >> /tmp/hook_debug.log

set -uo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
MAX_RETRIES=3
RETRY_STATE_FILE=".claude/hook_retry_count"
LOCK_FILE=".claude/hook.lock"
GOLANGCI_TIMEOUT="2m"

mkdir -p "$(dirname "$RETRY_STATE_FILE")"

# ---------------------------------------------------------------------------
# Read changed file paths from stdin (JSON) — expects a field like
# {"tool_input": {"file_path": "..."}} or {"files": ["...", "..."]}
# Adjust the jq filter below to match your actual Claude Code payload shape.
# ---------------------------------------------------------------------------
INPUT_JSON="$(cat)"

CHANGED_FILES="$(echo "$INPUT_JSON" | jq -r '
  if .tool_input.file_path then [.tool_input.file_path]
  elif .files then .files
  else []
  end | .[]' 2>/dev/null)"

if [[ -z "$CHANGED_FILES" ]]; then
  # Nothing we recognize as a Go file change — nothing to do.
  exit 0
fi

# Only care about Go files for this hook.
GO_FILES="$(echo "$CHANGED_FILES" | grep -E '\.go$' || true)"

if [[ -z "$GO_FILES" ]]; then
  exit 0
fi

# ---------------------------------------------------------------------------
# Retry limit guard — prevents infinite self-correction loops
# ---------------------------------------------------------------------------
RETRY_COUNT=0
if [[ -f "$RETRY_STATE_FILE" ]]; then
  RETRY_COUNT="$(cat "$RETRY_STATE_FILE")"
fi

if [[ "$RETRY_COUNT" -ge "$MAX_RETRIES" ]]; then
  echo "HOOK HALT: max retries ($MAX_RETRIES) reached for this task." >&2
  echo "Manual review required — the agent could not resolve the failing check." >&2
  rm -f "$RETRY_STATE_FILE"
  exit 2
fi

fail_and_increment() {
  local stage="$1"
  local output="$2"
  echo $((RETRY_COUNT + 1)) > "$RETRY_STATE_FILE"
  echo "=== HOOK FAILURE: $stage (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES) ===" >&2
  echo "$output" >&2
  exit 2
}

reset_retry_count() {
  rm -f "$RETRY_STATE_FILE"
}

# ---------------------------------------------------------------------------
# Locked file enforcement (optional but cheap — skip if you don't use it yet)
# List protected paths in .claude/locked_files.txt, one glob per line.
# ---------------------------------------------------------------------------
LOCKED_FILES_LIST=".claude/locked_files.txt"
if [[ -f "$LOCKED_FILES_LIST" ]]; then
  while IFS= read -r pattern; do
    [[ -z "$pattern" ]] && continue
    for f in $GO_FILES; do
      if [[ "$f" == $pattern ]]; then
        fail_and_increment "locked-file-check" \
          "File '$f' matches locked pattern '$pattern'. This file requires manual edit, not agent edit."
      fi
    done
  done < "$LOCKED_FILES_LIST"
fi

# ---------------------------------------------------------------------------
# 1. gofmt / goimports — cheapest, run first
# ---------------------------------------------------------------------------
FMT_OUT="$(gofmt -l $GO_FILES 2>&1)"
if [[ -n "$FMT_OUT" ]]; then
  # Auto-fix instead of just failing — formatting has no ambiguity.
  gofmt -w $GO_FILES
  echo "Auto-formatted with gofmt: $FMT_OUT"
fi

if command -v goimports >/dev/null 2>&1; then
  IMPORTS_OUT="$(goimports -l $GO_FILES 2>&1)"
  if [[ -n "$IMPORTS_OUT" ]]; then
    goimports -w $GO_FILES
    echo "Auto-fixed imports with goimports: $IMPORTS_OUT"
  fi
fi

# ---------------------------------------------------------------------------
# 2. go build — must compile before anything else matters
# ---------------------------------------------------------------------------
BUILD_OUT="$(go build ./... 2>&1)"
BUILD_STATUS=$?
if [[ $BUILD_STATUS -ne 0 ]]; then
  fail_and_increment "go build" "$BUILD_OUT"
fi

# ---------------------------------------------------------------------------
# 3. golangci-lint — only new/changed lines, using existing .golangci.yml
#    (depguard rules for Clean Architecture enforcement already configured)
# ---------------------------------------------------------------------------
if command -v golangci-lint >/dev/null 2>&1; then
  LINT_OUT="$(golangci-lint run --timeout "$GOLANGCI_TIMEOUT" --new-from-rev=HEAD 2>&1)"
  LINT_STATUS=$?
  if [[ $LINT_STATUS -ne 0 ]]; then
    fail_and_increment "golangci-lint" "$LINT_OUT"
  fi
else
  echo "WARNING: golangci-lint not found in PATH, skipping lint stage." >&2
fi

# ---------------------------------------------------------------------------
# 4. go test -race — only for packages containing changed files
# ---------------------------------------------------------------------------
CHANGED_PACKAGES="$(for f in $GO_FILES; do dirname "$f"; done | sort -u)"

TEST_FAILED=0
TEST_OUTPUT=""
for pkg in $CHANGED_PACKAGES; do
  if [[ -d "$pkg" ]]; then
    PKG_OUT="$(go test "./$pkg/..." -race -count=1 2>&1)"
    PKG_STATUS=$?
    if [[ $PKG_STATUS -ne 0 ]]; then
      TEST_FAILED=1
      TEST_OUTPUT="${TEST_OUTPUT}
--- $pkg ---
${PKG_OUT}"
    fi
  fi
done

if [[ $TEST_FAILED -ne 0 ]]; then
  fail_and_increment "go test -race" "$TEST_OUTPUT"
fi

# ---------------------------------------------------------------------------
# All checks passed
# ---------------------------------------------------------------------------
reset_retry_count
echo "All checks passed: gofmt, build, lint, test -race."
exit 0