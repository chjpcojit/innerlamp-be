#!/bin/bash
# check-gcp-config.sh
# Checks: GCP resource identifiers must not be hardcoded in source code
# Banking compliance: Multi-environment safety — hardcoded prod values in dev = data contamination risk
# Covers: Pub/Sub topics, Cloud Storage buckets, GCP project IDs, region strings, Secret Manager paths
# Referenced by: run-all.sh [16/16]

set -euo pipefail

ERRORS=()
DETAILS=()
HINTS=()

# ── Rule 1: No hardcoded GCP project IDs ─────────────────────────────────────
# Pattern: "innerlamp-prod" "innerlamp-staging" "innerlamp-dev" as string literals in Go files
GCP_PROJECT_PATTERN='"innerlamp-prod"\|"innerlamp-staging"\|"innerlamp-dev"\|"innerlamp-[a-z]*"'

PROJECT_VIOLATIONS=$(grep -rn "$GCP_PROJECT_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example\|comment\|\/\/" || true)

if [ -n "$PROJECT_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_GCP_PROJECT")
      DETAILS+=("Hardcoded GCP project ID in source code | File: $FILE:$LINENO")
      HINTS+=("Load from config: cfg.GCP.ProjectID — never hardcode project IDs. Wrong project in production = data written to wrong environment.")
    fi
  done <<< "$PROJECT_VIOLATIONS"
fi

# ── Rule 2: No hardcoded Pub/Sub topic names ──────────────────────────────────
# Pattern: "innerlamp.transaction.completed" or "projects/.../topics/..." literals
PUBSUB_PATTERN='"projects/[a-z-]*/topics/\|"innerlamp\.[a-z]*\.[a-z]*"'

PUBSUB_VIOLATIONS=$(grep -rn "$PUBSUB_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$PUBSUB_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_PUBSUB_TOPIC")
      DETAILS+=("Hardcoded Pub/Sub topic name | File: $FILE:$LINENO")
      HINTS+=("Load from config: cfg.PubSub.TransactionCompletedTopic. Topic names differ between dev/staging/prod environments.")
    fi
  done <<< "$PUBSUB_VIOLATIONS"
fi

# ── Rule 3: No hardcoded Cloud Storage bucket names ──────────────────────────
# Pattern: "innerlamp-documents-prod" "innerlamp-receipts-staging" etc.
BUCKET_PATTERN='"innerlamp-[a-z-]*-prod\|innerlamp-[a-z-]*-staging\|innerlamp-[a-z-]*-dev"'

BUCKET_VIOLATIONS=$(grep -rn "$BUCKET_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$BUCKET_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_GCS_BUCKET")
      DETAILS+=("Hardcoded Cloud Storage bucket name | File: $FILE:$LINENO")
      HINTS+=("Load from config: cfg.Storage.DocumentsBucket. Using prod bucket in dev = real user documents in test environment.")
    fi
  done <<< "$BUCKET_VIOLATIONS"
fi

# ── Rule 4: No hardcoded Secret Manager paths ────────────────────────────────
# Pattern: "projects/innerlamp-prod/secrets/..." as literal string
SECRET_PATTERN='"projects/innerlamp-[a-z]*/secrets/'

SECRET_VIOLATIONS=$(grep -rn "$SECRET_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$SECRET_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_SECRET_PATH")
      DETAILS+=("Hardcoded Secret Manager path | File: $FILE:$LINENO")
      HINTS+=("Use pkg/secrets/ with configurable project: secrets.Get(ctx, cfg.GCP.ProjectID, \"secret-name\"). Hardcoded prod secret path in dev code = accidental prod secret access.")
    fi
  done <<< "$SECRET_VIOLATIONS"
fi

# ── Rule 5: No hardcoded GCP region strings outside config ───────────────────
# Pattern: "us-central1" "asia-southeast1" as literals in non-config files
REGION_PATTERN='"us-central1"\|"asia-southeast1"\|"us-east1"\|"southamerica-east1"'

REGION_VIOLATIONS=$(grep -rn "$REGION_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$REGION_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_GCP_REGION")
      DETAILS+=("Hardcoded GCP region string | File: $FILE:$LINENO")
      HINTS+=("Load from config: cfg.GCP.Region. CNBV data residency rules may require specific regions — make it configurable per environment.")
    fi
  done <<< "$REGION_VIOLATIONS"
fi

# ── Rule 6: Cloud Run service URLs must not be hardcoded ─────────────────────
# Pattern: "https://users-service-xxx-uc.a.run.app" literal URLs
CLOUD_RUN_PATTERN='"https://[a-z-]*-[a-z0-9]*-[a-z]{2}\.a\.run\.app'

CLOUD_RUN_VIOLATIONS=$(grep -rn "$CLOUD_RUN_PATTERN" \
  --include="*.go" services/ \
  2>/dev/null \
  | grep -v "_test.go" \
  | grep -v "config\|Config\|example\|Example" || true)

if [ -n "$CLOUD_RUN_VIOLATIONS" ]; then
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      FILE=$(echo "$line" | cut -d: -f1)
      LINENO=$(echo "$line" | cut -d: -f2)
      ERRORS+=("HARDCODED_CLOUD_RUN_URL")
      DETAILS+=("Hardcoded Cloud Run service URL | File: $FILE:$LINENO")
      HINTS+=("Load from config: cfg.Services.UsersServiceURL. Cloud Run URLs change between dev/staging/prod deployments.")
    fi
  done <<< "$CLOUD_RUN_VIOLATIONS"
fi

# ── Export results ────────────────────────────────────────────────────────────
echo "GCP_CONFIG_ERRORS=${#ERRORS[@]}"
for i in "${!ERRORS[@]}"; do
  echo "ERR_TYPE_$i=${ERRORS[$i]}"
  echo "ERR_DETAIL_$i=${DETAILS[$i]}"
  echo "ERR_HINT_$i=${HINTS[$i]}"
done
