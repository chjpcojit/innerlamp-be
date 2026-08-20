#!/usr/bin/env bash
# scripts/check-readme-freshness.sh — PostToolUse (Write|Edit) hook.
#
# Per CLAUDE.md's per-folder README convention (pkg/<name>/, services/<svc>/,
# migrations/, scripts/): when a source file in one of those folders is
# written/edited, remind the model to check that folder's README.md if the
# source file is now newer than the README (or the README is missing).
#
# Heuristic (mtime comparison), not enforcement — never blocks the edit,
# only injects a reminder into the conversation via additionalContext.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

INPUT="$(cat)"
FILE="$(echo "$INPUT" | jq -r '.tool_response.filePath // .tool_input.file_path // empty')"

[ -z "$FILE" ] && exit 0
[[ "$FILE" == */README.md ]] && exit 0

case "$FILE" in
  "$REPO_ROOT"/*) REL="${FILE#"$REPO_ROOT"/}" ;;
  *) exit 0 ;;
esac

case "$REL" in
  pkg/*/*)      OWNER="pkg/$(echo "$REL" | cut -d/ -f2)" ;;
  services/*/*) OWNER="services/$(echo "$REL" | cut -d/ -f2)" ;;
  migrations/*) OWNER="migrations" ;;
  scripts/*)    OWNER="scripts" ;;
  *)            exit 0 ;;
esac

README="$REPO_ROOT/$OWNER/README.md"

if [ ! -f "$README" ]; then
  jq -n --arg owner "$OWNER" \
    '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("\($owner)/ has no README.md yet — add one per CLAUDE.md'"'"'s per-folder README convention before finishing this task.")}}'
  exit 0
fi

mtime() { stat -f %m "$1" 2>/dev/null || stat -c %Y "$1"; }

FILE_MTIME="$(mtime "$FILE")"
README_MTIME="$(mtime "$README")"

if [ "$FILE_MTIME" -gt "$README_MTIME" ]; then
  jq -n --arg rel "$REL" --arg owner "$OWNER" \
    '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("\($rel) changed after \($owner)/README.md was last touched — check whether \($owner)/README.md still matches before finishing this task.")}}'
fi

exit 0
