#!/bin/bash
# check-db-transactions.sh
# Checks: usecases with multiple write operations must use DB transactions
# Banking compliance: ACID required — partial writes cause data inconsistency
# Covers: PostgreSQL (pgx BeginTx) and MongoDB (session.WithTransaction)
# Referenced by: run-all.sh [14/16]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

USECASE_DIRS=$(find services/ -type d -name "usecase" 2>/dev/null | tr '\n' ' ')

if [ -z "$USECASE_DIRS" ]; then
  echo "DB_TX_ERRORS=0"
  echo "DB_TX_SKIPPED=no usecase directories found"
  exit 0
fi

# ── Rule 1: Usecase with ≥2 write calls must use a transaction ───────────────
# Write call patterns: .Create(, .Update(, .Save(, .Insert(, .Upsert(, .Delete(
WRITE_PATTERN="\.Create(\|\.Update(\|\.Save(\|\.Insert(\|\.Upsert(\|\.Delete("

# Transaction patterns: BeginTx, WithTransaction, session.StartTransaction
TX_PATTERN="BeginTx\|WithTransaction\|StartTransaction\|\.Begin(\|RunInTransaction"

for usecase_file in $(find $USECASE_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  # Count write operations in Execute function
  WRITE_COUNT=$(grep -c "$WRITE_PATTERN" "$usecase_file" 2>/dev/null || true)

  if [ "$WRITE_COUNT" -ge 2 ]; then
    # Check if transaction is used
    if ! grep -q "$TX_PATTERN" "$usecase_file" 2>/dev/null; then
      ERRORS+=("MISSING_DB_TRANSACTION")
      DETAILS+=("Usecase has $WRITE_COUNT write operations but no DB transaction | File: $usecase_file")
      HINTS+=("Wrap multiple write ops in a transaction. PostgreSQL: db.BeginTx(ctx, nil) then tx.Commit()/Rollback(). MongoDB: session.WithTransaction(ctx, func). If ops span both DBs, implement saga pattern with compensation.")
    fi
  fi
done

# ── Rule 2: PostgreSQL transactions must have defer Rollback ─────────────────
# Pattern: BeginTx used but no defer tx.Rollback — partial commit on panic
for usecase_file in $(find $USECASE_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  if grep -q "BeginTx\b" "$usecase_file" 2>/dev/null; then
    if ! grep -q "defer.*[Rr]ollback\b" "$usecase_file" 2>/dev/null; then
      ERRORS+=("MISSING_TX_ROLLBACK")
      DETAILS+=("BeginTx used without defer tx.Rollback() — transaction may leak on error | File: $usecase_file")
      HINTS+=("Always add: defer tx.Rollback() immediately after BeginTx. It is a no-op after Commit() so it is always safe.")
    fi
  fi
done

# ── Rule 3: MongoDB session must have defer session.EndSession ─────────────────
for usecase_file in $(find $USECASE_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
  if grep -q "StartSession\b\|client\.StartSession" "$usecase_file" 2>/dev/null; then
    if ! grep -q "defer.*EndSession\b\|defer.*session\.End" "$usecase_file" 2>/dev/null; then
      ERRORS+=("MISSING_MONGO_SESSION_END")
      DETAILS+=("MongoDB session started without defer session.EndSession() — connection leak | File: $usecase_file")
      HINTS+=("Add: defer session.EndSession(ctx) immediately after client.StartSession() to prevent connection pool exhaustion.")
    fi
  fi
done

# ── Rule 4: Repository layer must not manage transactions directly ─────────────
# Transaction ownership belongs to usecase layer, not repository
REPO_DIRS=$(find services/ -type d -name "repository" ! -path "*/domain/*" 2>/dev/null | tr '\n' ' ')

if [ -n "$REPO_DIRS" ]; then
  for repo_file in $(find $REPO_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null); do
    if grep -q "BeginTx\b\|\.Begin(\b" "$repo_file" 2>/dev/null; then
      ERRORS+=("TRANSACTION_IN_REPOSITORY")
      DETAILS+=("Repository layer is managing DB transactions — transaction ownership belongs to usecase | File: $repo_file")
      HINTS+=("Move BeginTx/Commit/Rollback to the usecase layer. Repository methods should accept an optional *sql.Tx or mongo.Session parameter injected from usecase.")
    fi
  done
fi

# ── Export results ────────────────────────────────────────────────────────────
echo "DB_TX_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
