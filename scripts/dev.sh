#!/usr/bin/env bash
#
# dev.sh — run Dust Almanac locally.
#
#   scripts/dev.sh            migrate + seed wooddust_dev, then serve on 127.0.0.1:$PORT
#   scripts/dev.sh --memory   no Postgres: serve data/ from memory (DATABASE_URL unset)
#
# The SPA is whatever was last built into web/dist (embedded at compile time).
# For hot reload, run `npm run dev` in web/ alongside this; Vite proxies
# /api, /images and /sitemap.xml to this server.
#
# Reads .env if present. Defaults: PORT=8080,
# DATABASE_URL=postgres://$USER@localhost/wooddust_dev?sslmode=disable.

set -euo pipefail
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"
if [ -f .env ]; then set -a; . ./.env; set +a; fi
export PORT="${PORT:-8080}"
export BASE_URL="${BASE_URL:-http://localhost:$PORT}"
mkdir -p web/dist && [ -e web/dist/.gitkeep ] || touch web/dist/.gitkeep

if [ "${1:-}" = "--memory" ]; then
  unset DATABASE_URL
  exec go run ./cmd/wooddust serve
fi

export DATABASE_URL="${DATABASE_URL:-postgres://${USER}@localhost/wooddust_dev?sslmode=disable}"
if command -v createdb >/dev/null 2>&1; then
  createdb wooddust_dev 2>/dev/null || true
fi
go build -o bin/wooddust ./cmd/wooddust
bin/wooddust seed
exec bin/wooddust serve
