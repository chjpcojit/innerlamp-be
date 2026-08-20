#!/usr/bin/env bash
#
# fast_check.sh
# PostToolUse hook (matcher: Edit|Write) for Go projects.
# Runs on EVERY file edit. Keep this cheap — only checks that must always
# hold true, even mid-TDD-cycle (build must always compile).
# Does NOT run tests or lint here — see full_check.sh (Stop hook) for those.
#
# Exit code contract:
#   0 -> ok, agent continues
#   2 -> blocked, stderr fed back to agent as feedback

set -uo pipefail

MAX_RETRIES=3
RETRY_STATE_FILE=".claude/fast_check_retry_count"
LOG_FILE=".claude/hook_log.jsonl"

mkdir -p "$(dirname "$RETRY_STATE_FILE")"

# log_event <result> <stage> <detail>
# Appends one compact JSON line. Best-effort — logging must never break the hook.
log_event() {
  local result="$1" stage="$2" detail="${3:-}"
  jq -nc \
    --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg hook "fast_check" \
    --arg file "${FILE_PATH:-}" \
    --arg result "$result" \
    --arg stage "$stage" \
    --arg detail "$detail" \
    --argjson retry "${RETRY_COUNT:-0}" \
    '{timestamp:$ts, hook:$hook, file:$file, result:$result, stage:$stage, retry_count:$retry, detail:$detail}' \
    >> "$LOG_FILE" 2>/dev/null || true
}

INPUT_JSON="$(cat)"
FILE_PATH="$(echo "$INPUT_JSON" | jq -r '.tool_input.file_path // empty')"

# Only care about Go files.
if [[ -z "$FILE_PATH" || "$FILE_PATH" != *.go ]]; then
  exit 0
fi

RETRY_COUNT=0
if [[ -f "$RETRY_STATE_FILE" ]]; then
  RETRY_COUNT="$(cat "$RETRY_STATE_FILE")"
fi

if [[ "$RETRY_COUNT" -ge "$MAX_RETRIES" ]]; then
  echo "FAST_CHECK HALT: max retries ($MAX_RETRIES) reached on $FILE_PATH." >&2
  echo "Manual review required — build keeps failing after repeated fixes." >&2
  log_event "halt" "go build" "max retries reached"
  rm -f "$RETRY_STATE_FILE"
  exit 2
fi

fail_and_increment() {
  local stage="$1"
  local output="$2"
  echo $((RETRY_COUNT + 1)) > "$RETRY_STATE_FILE"
  echo "=== FAST_CHECK FAILURE: $stage (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES) ===" >&2
  echo "$output" >&2
  log_event "fail" "$stage" "$(printf '%s' "$output" | head -c 300)"
  exit 2
}

# --- 1. gofmt (auto-fix, never blocks) ---
gofmt -w "$FILE_PATH" 2>/dev/null

# --- 2. goimports (auto-fix, never blocks) ---
if command -v goimports >/dev/null 2>&1; then
  goimports -w "$FILE_PATH" 2>/dev/null
fi

# --- 3. go build (the only thing allowed to block here) ---
BUILD_OUT="$(go build ./... 2>&1)"
BUILD_STATUS=$?
if [[ $BUILD_STATUS -ne 0 ]]; then
  fail_and_increment "go build" "$BUILD_OUT"
fi

# Passed — reset retry count for this file-level check.
rm -f "$RETRY_STATE_FILE"
log_event "pass" "go build" ""
exit 0