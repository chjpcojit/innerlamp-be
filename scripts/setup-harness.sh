#!/bin/bash
# setup-harness.sh
# Installs all tools required by the InnerLamp Banking Harness constraint suite.
# Run once on a new machine or CI environment.
# Usage: ./scripts/setup-harness.sh

set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

MISSING=()

log_ok()   { echo -e "${GREEN}[OK]${NC}   $1"; }
log_skip() { echo -e "${YELLOW}[SKIP]${NC} $1 (already installed)"; }
log_err()  { echo -e "${RED}[FAIL]${NC} $1"; MISSING+=("$1"); }

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║   InnerLamp Harness — Tool Setup                    ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# ── 1. Go (required baseline) ─────────────────────────────────────────────────
if command -v go &>/dev/null; then
  log_skip "go ($(go version | awk '{print $3}'))"
else
  log_err "go — install from https://go.dev/dl/ then re-run this script"
fi

# ── 2. gosec (static security analysis) ──────────────────────────────────────
if command -v gosec &>/dev/null; then
  log_skip "gosec ($(gosec --version 2>&1 | head -1))"
else
  echo "Installing gosec..."
  go install github.com/securego/gosec/v2/cmd/gosec@latest && log_ok "gosec" || log_err "gosec"
fi

# ── 3. govulncheck (dependency vulnerability scanning) ────────────────────────
if command -v govulncheck &>/dev/null; then
  log_skip "govulncheck"
else
  echo "Installing govulncheck..."
  go install golang.org/x/vuln/cmd/govulncheck@latest && log_ok "govulncheck" || log_err "govulncheck"
fi

# ── 4. jq (JSON parsing in security scanner) ─────────────────────────────────
if command -v jq &>/dev/null; then
  log_skip "jq ($(jq --version))"
else
  OS="$(uname -s)"
  echo "Installing jq..."
  case "$OS" in
    Darwin)  brew install jq && log_ok "jq" || log_err "jq" ;;
    Linux)   sudo apt-get install -y jq && log_ok "jq" || log_err "jq" ;;
    *)       log_err "jq — install manually from https://jqlang.github.io/jq/" ;;
  esac
fi

# ── 5. bc (arbitrary precision calculator, used in coverage checks) ───────────
if command -v bc &>/dev/null; then
  log_skip "bc"
else
  OS="$(uname -s)"
  echo "Installing bc..."
  case "$OS" in
    Darwin)  brew install bc && log_ok "bc" || log_err "bc" ;;
    Linux)   sudo apt-get install -y bc && log_ok "bc" || log_err "bc" ;;
    *)       log_err "bc — install manually for your OS" ;;
  esac
fi

# ── 6. golangci-lint (optional but used in Makefile) ─────────────────────────
if command -v golangci-lint &>/dev/null; then
  log_skip "golangci-lint ($(golangci-lint --version 2>&1 | head -1))"
else
  echo "Installing golangci-lint..."
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" latest \
    && log_ok "golangci-lint" || log_err "golangci-lint (optional)"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
if [ ${#MISSING[@]} -eq 0 ]; then
  echo -e "${GREEN}All harness tools installed. Run: ./scripts/run-all.sh${NC}"
  echo ""
else
  echo -e "${RED}${#MISSING[@]} tool(s) failed to install:${NC}"
  for t in "${MISSING[@]}"; do
    echo "  - $t"
  done
  echo ""
  echo "Fix the above tools before running ./scripts/run-all.sh"
  exit 1
fi
