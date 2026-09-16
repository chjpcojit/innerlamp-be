#!/usr/bin/env bash
#
# protect_files.sh
# PreToolUse hook (matcher: Edit|Write) for Go projects.
# Blocks the tool call BEFORE it executes if the target file matches any
# pattern in .claude/locked_files.txt. Unlike fast_check.sh/full_check.sh,
# this hook does not need retry logic — a locked file is an absolute
# boundary, not something the agent should ever "fix its way past".
#
# Exit code contract:
#   0 -> not locked, proceed as normal
#   2 -> blocked, stderr fed back to Claude as the reason

set -uo pipefail

# Anchor every path to the project root, never the current working directory.
# Otherwise running from a subdirectory creates a stray .claude/ folder there.
ROOT="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[[ -z "$ROOT" ]] && exit 0

LOCKED_FILES_LIST="$ROOT/.claude/locked_files.txt"
LOG_FILE="$ROOT/.claude/hook_log.jsonl"

# log_event <result> <detail>
# Best-effort — logging must never break the hook.
log_event() {
  local result="$1" detail="${2:-}"
  jq -nc \
    --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg hook "protect_files" \
    --arg file "${FILE_PATH:-}" \
    --arg result "$result" \
    --arg detail "$detail" \
    '{timestamp:$ts, hook:$hook, file:$file, result:$result, detail:$detail}' \
    >> "$LOG_FILE" 2>/dev/null || true
}

INPUT_JSON="$(cat)"
FILE_PATH="$(echo "$INPUT_JSON" | jq -r '.tool_input.file_path // empty')"

# No file path in this tool call (shouldn't happen for Edit/Write, but guard anyway).
if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

# Normalize Windows-style separators just in case.
FILE_PATH="${FILE_PATH//\\//}"

# No config file yet -> nothing is locked. Fails open by design: an empty
# or missing list means this hook is a no-op, not a hard block.
if [[ ! -f "$LOCKED_FILES_LIST" ]]; then
  exit 0
fi

while IFS= read -r pattern; do
  # Skip blank lines and comments (lines starting with #).
  [[ -z "$pattern" || "$pattern" == \#* ]] && continue

  if [[ "$FILE_PATH" == $pattern ]]; then
    echo "BLOCKED: '$FILE_PATH' matches locked pattern '$pattern'." >&2
    echo "This file requires manual edit by a human — not an agent edit." >&2
    echo "If this file genuinely needs to change, ask the user to edit it directly," >&2
    echo "or to remove/adjust the pattern in $LOCKED_FILES_LIST." >&2
    log_event "blocked" "matched pattern: $pattern"
    exit 2
  fi
done < "$LOCKED_FILES_LIST"

# Only log the happy path for Go files — otherwise every .md edit appends a
# useless "pass" line and the log fills with noise.
if [[ "$FILE_PATH" == *.go ]]; then
  log_event "pass" ""
fi
exit 0