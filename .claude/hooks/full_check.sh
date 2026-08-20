#!/usr/bin/env bash
#
# full_check.sh
# Stop hook for Go projects. Fires when Claude finishes a turn.
# Runs the expensive checks: golangci-lint + go test -race.
# Uses "ok"/"reason" JSON output (see Claude Code prompt/agent hook contract)
# so a failure feeds back as an instruction, not a hard stop.
#
# IMPORTANT: Stop hooks fire whenever Claude finishes RESPONDING, not only
# at full task completion. Claude Code caps consecutive Stop-hook blocks at 8
# (CLAUDE_CODE_STOP_HOOK_BLOCK_CAP) — this script adds its own tighter limit
# on top, and always checks stop_hook_active to avoid re-triggering itself.

set -uo pipefail

MAX_RETRIES=3
RETRY_STATE_FILE=".claude/full_check_retry_count"
LOG_FILE=".claude/hook_log.jsonl"
GOLANGCI_TIMEOUT="3m"

mkdir -p "$(dirname "$RETRY_STATE_FILE")"

# log_event <result> <stage> <detail>
# Best-effort — logging must never break the hook.
log_event() {
  local result="$1" stage="$2" detail="${3:-}"
  jq -nc \
    --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg hook "full_check" \
    --arg result "$result" \
    --arg stage "$stage" \
    --arg detail "$detail" \
    --argjson retry "${RETRY_COUNT:-0}" \
    '{timestamp:$ts, hook:$hook, result:$result, stage:$stage, retry_count:$retry, detail:$detail}' \
    >> "$LOG_FILE" 2>/dev/null || true
}

INPUT_JSON="$(cat)"

# --- Guard against re-triggering our own continuation loop ---
STOP_HOOK_ACTIVE="$(echo "$INPUT_JSON" | jq -r '.stop_hook_active // false')"
if [[ "$STOP_HOOK_ACTIVE" == "true" ]]; then
  # Claude Code is already re-invoking us because we blocked last time.
  # Do NOT block again without checking retry count below — fall through.
  :
fi

RETRY_COUNT=0
if [[ -f "$RETRY_STATE_FILE" ]]; then
  RETRY_COUNT="$(cat "$RETRY_STATE_FILE")"
fi

if [[ "$RETRY_COUNT" -ge "$MAX_RETRIES" ]]; then
  echo "FULL_CHECK: max retries ($MAX_RETRIES) reached — letting Claude stop." >&2
  echo "Lint/test still failing. Manual review required." >&2
  log_event "halt" "retry-cap" "max retries reached, allowing stop"
  rm -f "$RETRY_STATE_FILE"
  # Allow the stop — do NOT keep blocking past the retry budget.
  echo '{"ok": true}'
  exit 0
fi

block_and_increment() {
  local reason="$1"
  local stage="$2"
  echo $((RETRY_COUNT + 1)) > "$RETRY_STATE_FILE"
  log_event "fail" "$stage" "$(printf '%s' "$reason" | head -c 300)"
  # Escape for JSON via jq -Rs
  local escaped
  escaped="$(printf '%s' "$reason" | jq -Rs .)"
  echo "{\"ok\": false, \"reason\": $escaped}"
  exit 0
}

# ---------------------------------------------------------------------------
# TDD awareness: if every .go file touched this session is a _test.go file
# (i.e. no implementation file changed yet), a failing `go test` is expected
# — this is the "red" phase, not a real failure. Don't block on it.
# Heuristic only: relies on git to see what's modified/untracked.
# ---------------------------------------------------------------------------
CHANGED_GO_FILES="$(git status --porcelain 2>/dev/null | awk '{print $2}' | grep -E '\.go$' || true)"
NON_TEST_FILES="$(echo "$CHANGED_GO_FILES" | grep -v '_test\.go$' || true)"

ONLY_TESTS_CHANGED=false
if [[ -n "$CHANGED_GO_FILES" && -z "$NON_TEST_FILES" ]]; then
  ONLY_TESTS_CHANGED=true
fi

# ---------------------------------------------------------------------------
# 1. golangci-lint (full run — existing .golangci.yml with depguard rules)
# ---------------------------------------------------------------------------
if command -v golangci-lint >/dev/null 2>&1; then
  LINT_OUT="$(golangci-lint run --timeout "$GOLANGCI_TIMEOUT" 2>&1)"
  LINT_STATUS=$?
  if [[ $LINT_STATUS -ne 0 ]]; then
    block_and_increment "golangci-lint failed:
${LINT_OUT}" "golangci-lint"
  fi
fi

# ---------------------------------------------------------------------------
# 2. go test -race
# ---------------------------------------------------------------------------
TEST_OUT="$(go test ./... -race -count=1 2>&1)"
TEST_STATUS=$?

if [[ $TEST_STATUS -ne 0 ]]; then
  if [[ "$ONLY_TESTS_CHANGED" == "true" ]]; then
    # Expected red phase of TDD — do not block, just remind the agent.
    log_event "pass" "go test -race" "TDD red phase, not blocked"
    echo '{"ok": true}'
    exit 0
  fi
  block_and_increment "go test -race failed:
${TEST_OUT}" "go test -race"
fi

# All checks passed — reset retry count.
rm -f "$RETRY_STATE_FILE"
log_event "pass" "golangci-lint+test" ""
echo '{"ok": true}'
exit 0