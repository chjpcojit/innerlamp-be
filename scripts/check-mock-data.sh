#!/bin/bash
# check-mock-data.sh
# Kiểm tra mock data và TODO chưa implement trong production code.
#
# Checks:
#   1. TODO / FIXME comments in non-test, non-example Go source files
#   2. MOCK_DATA_MARKER annotations in example files that lack a NOT_FOR_PRODUCTION header
#   3. Mock/fake/stub patterns in production code (services/ and pkg/ outside examples/)
#   4. Hardcoded in-memory credential maps (e.g. `var users = map[string]*User{...}`)
#
# Scope:
#   - services/**/*.go           (excluding *_test.go)
#   - pkg/**/*.go               (excluding *_test.go and examples/**)
#
# Exit 0 always (violations reported via structured output for run-all.sh)

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Helper: record a violation ────────────────────────────────────────────────
add_violation() {
  local type="$1"
  local detail="$2"
  local hint="$3"
  ERRORS+=("$type")
  DETAILS+=("$detail")
  HINTS+=("$hint")
}

# ── Rule 1: TODO / FIXME in production service code ───────────────────────────
# Scope: services/**/*.go  (not _test.go)
# Skip lines that are part of a MOCK_SERVICE block (they are reported by Rule 6)
while IFS= read -r match; do
  [ -z "$match" ] && continue
  file=$(echo "$match" | cut -d: -f1)
  lineno=$(echo "$match" | cut -d: -f2)
  text=$(echo "$match" | cut -d: -f3-)
  # If this is a TODO_PROD line and the file already has a MOCK_SERVICE marker,
  # skip it — Rule 6 will report the whole file with full context.
  if echo "$text" | grep -q "TODO_PROD:" && grep -q "MOCK_SERVICE:" "$file" 2>/dev/null; then
    continue
  fi
  add_violation \
    "UNFINISHED_TODO" \
    "TODO/FIXME comment in production code | File: $file:$lineno | Code: $text" \
    "Resolve the TODO before merging to production. If intentional, track in Jira and remove from code."
done < <(grep -rn "//\s*TODO\|//\s*FIXME\|//\s*HACK\|//\s*XXX" \
  --include="*.go" services/ 2>/dev/null | grep -v "_test.go" || true)

# ── Rule 2: TODO / FIXME in non-example pkg code ──────────────────────────────
while IFS= read -r match; do
  [ -z "$match" ] && continue
  # Skip example directories (they are allowed to have TODOs for illustration)
  echo "$match" | grep -q "/examples/" && continue
  file=$(echo "$match" | cut -d: -f1)
  lineno=$(echo "$match" | cut -d: -f2)
  text=$(echo "$match" | cut -d: -f3-)
  add_violation \
    "UNFINISHED_TODO" \
    "TODO/FIXME comment in production pkg code | File: $file:$lineno | Code: $text" \
    "Resolve the TODO before merging to production. If intentional, track in Jira and remove from code."
done < <(grep -rn "//\s*TODO\|//\s*FIXME\|//\s*HACK\|//\s*XXX" \
  --include="*.go" pkg/ 2>/dev/null | grep -v "_test.go" || true)

# ── Rule 3: Mock/fake/stub patterns in production service code ────────────────
# Detect: comment lines that say "Mock" or variable names suggesting fake data
# Exclude: test files and lines that reference testify/mock (test infrastructure)
while IFS= read -r match; do
  [ -z "$match" ] && continue
  # Skip testify/mock import lines
  echo "$match" | grep -qE '"github.com/stretchr/testify/mock"' && continue
  file=$(echo "$match" | cut -d: -f1)
  lineno=$(echo "$match" | cut -d: -f2)
  text=$(echo "$match" | cut -d: -f3-)
  add_violation \
    "MOCK_IN_PRODUCTION" \
    "Mock/fake/stub pattern found in production code | File: $file:$lineno | Code: $text" \
    "Replace mock implementation with a real implementation. See pkg/ for available clients (pkg/sms/, pkg/email/, etc.)."
done < <(grep -rn \
  "//\s*[Mm]ock\s\|//\s*[Ff]ake\s\|//\s*[Ss]tub\s\|MOCK_DATA_MARKER\|NOT_IMPLEMENTED_PLACEHOLDER\|panic(\"not implemented\")" \
  --include="*.go" services/ 2>/dev/null | grep -v "_test.go" || true)

# ── Rule 4: Example files must carry NOT_FOR_PRODUCTION header ────────────────
# Any .go file under an examples/ directory that has a MOCK_DATA_MARKER but
# is missing the ⚠️ MOCK DATA - NOT FOR PRODUCTION header comment.
while IFS= read -r example_file; do
  [ -z "$example_file" ] && continue
  # Check if this example file contains MOCK_DATA_MARKER
  if grep -q "MOCK_DATA_MARKER\|// Mock\|// Fake\|// Stub\|plaintext.*password\|// In production" \
      "$example_file" 2>/dev/null; then
    # It must also have the NOT FOR PRODUCTION header
    if ! grep -q "NOT FOR PRODUCTION\|NOT_FOR_PRODUCTION" "$example_file" 2>/dev/null; then
      add_violation \
        "EXAMPLE_MISSING_MOCK_WARNING" \
        "Example file contains mock data but lacks ⚠️ NOT FOR PRODUCTION header | File: $example_file" \
        "Add a file-level comment block: '// ⚠️ MOCK DATA - NOT FOR PRODUCTION' and list what needs replacing. See pkg/jwt/examples/http/main.go for reference."
    fi
  fi
done < <(find pkg/ services/ -path "*/examples/*.go" ! -name "*_test.go" 2>/dev/null || true)

# ── Rule 5: Hardcoded in-memory credential stores in non-test/non-example code ─
# Detect patterns like:  var <name> = map[string]*<Type>{ ... email: ... password: ... }
# This is the classic "fake auth database" anti-pattern.
while IFS= read -r match; do
  [ -z "$match" ] && continue
  echo "$match" | grep -q "/examples/" && continue
  file=$(echo "$match" | cut -d: -f1)
  lineno=$(echo "$match" | cut -d: -f2)
  text=$(echo "$match" | cut -d: -f3-)
  add_violation \
    "HARDCODED_AUTH_MAP" \
    "Potential hardcoded credential map in production code | File: $file:$lineno | Code: $text" \
    "Replace with a real user repository (pkg/database/ + Cloud SQL/MongoDB). Never store credentials inline."
done < <(grep -rn 'var [a-zA-Z]* = map\[string\]\*' \
  --include="*.go" services/ pkg/ 2>/dev/null \
  | grep -v "_test.go" | grep -i "user\|cred\|auth\|account" || true)

# ── Rule 6: MOCK_SERVICE markers — pending production implementations ──────────
# Detect: // ⚠️ MOCK_SERVICE: <name> placed by TDD Implementer or Infrastructure Task Generator
# These mark infrastructure adapters that are stubs needing real provider integration.
# Scope: services/**/*.go  (excluding _test.go)
MOCK_SEEN_FILES=""

while IFS= read -r file; do
  [ -z "$file" ] && continue

  # Deduplicate: one violation per file
  if echo "$MOCK_SEEN_FILES" | grep -qF "$file"; then
    continue
  fi
  MOCK_SEEN_FILES="$MOCK_SEEN_FILES|$file"

  # Extract service name(s) from MOCK_SERVICE: lines in this file
  service_names=$(grep -h "MOCK_SERVICE:" "$file" \
    | sed 's/.*MOCK_SERVICE:[[:space:]]*//' \
    | tr '\n' ',' | sed 's/,$//' || true)

  # Extract Provider, Reason, TODO_PROD, Tracker if present
  provider=$(grep -h "// Provider:" "$file" 2>/dev/null | head -1 \
    | sed 's/.*\/\/ Provider:[[:space:]]*//' || echo "unknown")
  reason=$(grep -h "// Reason:" "$file" 2>/dev/null | head -1 \
    | sed 's/.*\/\/ Reason:[[:space:]]*//' || echo "see file")
  todo_prod=$(grep -h "// TODO_PROD:" "$file" 2>/dev/null | head -1 \
    | sed 's/.*\/\/ TODO_PROD:[[:space:]]*//' || echo "see file")
  tracker=$(grep -h "// Tracker:" "$file" 2>/dev/null | head -1 \
    | sed 's/.*\/\/ Tracker:[[:space:]]*//' || echo "no tracker")

  add_violation \
    "MOCK_SERVICE_PENDING" \
    "Mock service adapter needs real implementation | Service: ${service_names} | File: ${file} | Provider: ${provider} | Reason: ${reason} | TODO_PROD: ${todo_prod} | Tracker: ${tracker}" \
    "Implement the real provider integration. See TODO_PROD comment in the file. Remove the MOCK_SERVICE header once real integration is tested and deployed."
done < <(grep -rl "MOCK_SERVICE:" --include="*.go" services/ 2>/dev/null \
  | grep -v "_test.go" || true)

# ── Output structured results ─────────────────────────────────────────────────
echo "MOCKDATA_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
