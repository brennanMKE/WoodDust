#!/usr/bin/env bash
#
# check.sh — the canonical verification run. Use it instead of hand-rolled
# go test invocations so everyone verifies the same way.
#
#   scripts/check.sh          go vet + go test + JSON Schema validation + seed integrity
#   scripts/check.sh go       go vet + go test only
#   scripts/check.sh data     ajv schema validation + `wooddust seed -check` only
#
# DB-backed tests (internal/store) SKIP unless TEST_DATABASE_URL is set. This
# script defaults it to the local wooddust_test database when psql can reach
# it, and says loudly when it cannot, because a green run without it does not
# cover SQL. The DB tests migrate that database down and up: never point
# TEST_DATABASE_URL at data you care about.

set -uo pipefail
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"
MODE="${1:-all}"
FAILED=0

step() { printf '\n== %s\n' "$*"; }
fail() { echo "FAIL: $*"; FAILED=1; }

# //go:embed all:dist needs at least one file; Vite's emptyOutDir removes it.
mkdir -p web/dist && [ -e web/dist/.gitkeep ] || touch web/dist/.gitkeep

run_go() {
  if [ -z "${TEST_DATABASE_URL:-}" ] && command -v psql >/dev/null 2>&1 \
     && psql -d wooddust_test -Atc 'select 1' >/dev/null 2>&1; then
    export TEST_DATABASE_URL="postgres://${USER}@localhost/wooddust_test?sslmode=disable"
  fi
  if [ -z "${TEST_DATABASE_URL:-}" ]; then
    echo "WARNING: TEST_DATABASE_URL unset — DB-backed tests will SKIP."
  fi
  step "gofmt"
  unformatted="$(gofmt -l cmd internal migrations web/*.go)"
  [ -z "$unformatted" ] || fail "gofmt needed: $unformatted"
  step "go vet"
  go vet ./... || fail "go vet"
  step "go test"
  # -p 1: the store tests share one database.
  go test -p 1 -count=1 ./... 2>&1 | tail -40
  [ "${PIPESTATUS[0]}" -eq 0 ] || fail "go test"
}

run_data() {
  step "JSON Schema (ajv)"
  if command -v npx >/dev/null 2>&1; then
    npx --yes -p ajv-cli@5 -p ajv-formats@3 ajv validate --spec=draft2020 -c ajv-formats \
      -s data/schema/species.schema.json -d "data/species/*.json" 2>&1 | grep -v ' valid$' | tail -20
    [ "${PIPESTATUS[0]}" -eq 0 ] || fail "species schema"
    npx --yes -p ajv-cli@5 -p ajv-formats@3 ajv validate --spec=draft2020 -c ajv-formats \
      -s data/schema/sources.schema.json -d data/sources.json 2>&1 | tail -5
    [ "${PIPESTATUS[0]}" -eq 0 ] || fail "sources schema"
  else
    fail "npx not found; cannot run ajv-cli"
  fi
  step "seed integrity (dry run)"
  go run ./cmd/wooddust seed -check || fail "seed -check"
}

case "$MODE" in
  go) run_go ;;
  data) run_data ;;
  all) run_go; run_data ;;
  *) echo "usage: $0 [all|go|data]"; exit 2 ;;
esac

echo
if [ "$FAILED" -ne 0 ]; then echo "VERIFICATION FAILED"; exit 1; fi
echo "VERIFICATION PASSED"
