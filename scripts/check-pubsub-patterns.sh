#!/bin/bash
# check-pubsub-patterns.sh
# Checks: GCP Cloud Pub/Sub publisher and subscriber usage patterns
#
# WHY: Pub/Sub is central to InnerLamp's event-driven architecture
# (innerlamp.transaction.completed, innerlamp.account.created, etc.).
# Incorrect patterns cause:
#   - Message loss (Nack missing → message redelivered infinitely)
#   - Phantom events (Publish before DB commit → event emitted even if DB rolls back)
#   - Silent message leaks (Publish without error check)
#   - Naming drift (topic names not following innerlamp.<domain>.<event> convention)
#
# RULES:
#   Rule 1 — Topic NAME CONSTANTS must follow: innerlamp.<domain>.<event>
#   Rule 2 — Subscriber Receive() handlers must call msg.Ack() AND msg.Nack()
#   Rule 3 — Publishers must not publish BEFORE a DB commit in same function
#   Rule 4 — Publish() calls must check the returned error (no bare _ = publish)
#   Rule 5 — Subscriber handlers must have context.WithTimeout or respect ctx.Done
#
# Banking compliance: Lost events = undetected failed transactions.
# Phantom events = ghost transactions reported to CNBV.
# Referenced by: run-all.sh [19/21]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Detect files that use Pub/Sub ────────────────────────────────────────────
PUBSUB_FILES=$(grep -rl \
  -e '"cloud.google.com/go/pubsub"' \
  -e 'pubsub.Client' \
  -e 'pubsub.Topic' \
  -e 'pubsub.Subscription' \
  --include="*.go" \
  services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -z "$PUBSUB_FILES" ]; then
  # No Pub/Sub usage yet — skip (not an error, feature may not be implemented)
  echo "PUBSUB_ERRORS=0"
  exit 0
fi

# ── Rule 1: Topic name constants must follow innerlamp.<domain>.<event> ─────────
# Detect string literals used as topic names that DON'T match the convention
TOPIC_VIOLATIONS=$(grep -rn \
  '"[a-z][a-z-]*\.[a-z][a-z-]*"' \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v 'innerlamp\.' \
  | grep -i 'topic\|Topic\|TOPIC\|pubsub' || true)

if [ -n "$TOPIC_VIOLATIONS" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    LINENO=$(echo "$line" | cut -d: -f2)
    CODE=$(echo "$line" | cut -d: -f3-)
    ERRORS+=("PUBSUB_TOPIC_NAMING")
    DETAILS+=("Pub/Sub topic name does not follow innerlamp.<domain>.<event> convention | File: $FILE:$LINENO | Code: $CODE")
    HINTS+=("Use naming convention: innerlamp.<domain>.<event>. Examples: innerlamp.transaction.completed, innerlamp.account.created, innerlamp.user.verified. Load from config, not as inline string literal.")
  done <<< "$TOPIC_VIOLATIONS"
fi

# ── Rule 2: Subscriber Receive() handlers must call both Ack and Nack ────────
for pubsub_file in $PUBSUB_FILES; do
  # Only check files that have .Receive( — subscriber files
  if ! grep -q '\.Receive(' "$pubsub_file" 2>/dev/null; then
    continue
  fi

  has_ack=$(grep -c '\.Ack()' "$pubsub_file" 2>/dev/null || echo "0")
  has_nack=$(grep -c '\.Nack()' "$pubsub_file" 2>/dev/null || echo "0")

  if [ "$has_ack" -eq 0 ]; then
    ERRORS+=("PUBSUB_MISSING_ACK")
    DETAILS+=("Pub/Sub subscriber has Receive() but no msg.Ack() | File: $pubsub_file")
    HINTS+=("After successful message processing call msg.Ack(). Without Ack, the subscription will redeliver the message up to ackDeadline times, causing duplicate event processing.")
  fi

  if [ "$has_nack" -eq 0 ]; then
    ERRORS+=("PUBSUB_MISSING_NACK")
    DETAILS+=("Pub/Sub subscriber has Receive() but no msg.Nack() | File: $pubsub_file")
    HINTS+=("On processing error call msg.Nack() so the message is immediately re-queued. Without Nack, failed messages sit until ackDeadline expires, delaying retry and causing SLA violations.")
  fi
done

# ── Rule 3: Publish must NOT happen before Commit in same function ─────────────
# Heuristic: in same function body, .Publish( appears before .Commit(
# This is a best-effort grep scan for co-located publish+commit in same file
for pubsub_file in $PUBSUB_FILES; do
  if ! grep -q '\.Publish(' "$pubsub_file" 2>/dev/null; then
    continue
  fi

  # Check if same file also has DB commit operations
  if grep -qE '\.Commit(|\.CommitTx(' "$pubsub_file" 2>/dev/null; then
    # Get line numbers of Publish and Commit
    publish_line=$(grep -n '\.Publish(' "$pubsub_file" 2>/dev/null | head -1 | cut -d: -f1 || echo "0")
    commit_line=$(grep -n '\.Commit(' "$pubsub_file" 2>/dev/null | head -1 | cut -d: -f1 || echo "99999")

    if [ "$publish_line" -lt "$commit_line" ] && [ "$publish_line" -ne "0" ]; then
      ERRORS+=("PUBSUB_PUBLISH_BEFORE_COMMIT")
      DETAILS+=("Publish() appears before DB Commit() in same file | File: $pubsub_file | Publish at line $publish_line, Commit at line $commit_line")
      HINTS+=("Always publish events AFTER the DB transaction commits. Pattern: 1) Begin tx, 2) Write to DB, 3) Commit tx, 4) Publish event. If Publish fails after Commit, use an outbox table pattern for guaranteed delivery.")
    fi
  fi
done

# ── Rule 4: Publish() return value (ServerID, error) must not be discarded ───
PUBLISH_IGNORED=$(grep -rn \
  -e '^\s*_\s*=.*\.Publish(' \
  -e '^\s*_,\s*_\s*=.*\.Publish(' \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" || true)

if [ -n "$PUBLISH_IGNORED" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    FILE=$(echo "$line" | cut -d: -f1)
    LINENO=$(echo "$line" | cut -d: -f2)
    ERRORS+=("PUBSUB_IGNORED_PUBLISH_ERROR")
    DETAILS+=("Publish() result discarded with _ | File: $FILE:$LINENO")
    HINTS+=("Check both return values: result, err := topic.Publish(ctx, msg).Get(ctx). A discarded error means silent message loss in production — a banking incident waiting to happen.")
  done <<< "$PUBLISH_IGNORED"
fi

# ── Rule 5: Subscriber handlers must respect context cancellation ─────────────
for pubsub_file in $PUBSUB_FILES; do
  if ! grep -q '\.Receive(' "$pubsub_file" 2>/dev/null; then
    continue
  fi

  # Check if there's any ctx.Done() or context.WithTimeout near Receive
  has_ctx_handling=$(grep -c \
    -e 'ctx\.Done()' \
    -e 'context\.WithTimeout' \
    -e 'context\.WithDeadline' \
    -e 'context\.WithCancel' \
    "$pubsub_file" 2>/dev/null || echo "0")

  if [ "$has_ctx_handling" -eq 0 ]; then
    ERRORS+=("PUBSUB_NO_CTX_DEADLINE")
    DETAILS+=("Subscriber Receive() handler has no context timeout/cancellation | File: $pubsub_file")
    HINTS+=("Wrap Receive() with context.WithTimeout: ctx, cancel := context.WithTimeout(ctx, 30*time.Second); defer cancel(). Without a deadline, a slow external dependency can cause the subscription handler to hang indefinitely, blocking all message processing.")
  fi
done

# ── Export results ────────────────────────────────────────────────────────────
echo "PUBSUB_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
