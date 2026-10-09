#!/usr/bin/env bash
# Start the app locally: checks MySQL, the database and the port, then
# builds and runs the server. Usage: scripts/start.sh   (or: make start)
set -euo pipefail

cd "$(dirname "$0")/.."

say()  { printf '%s\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

# 1. Settings.
[ -f .env ] || fail ".env is missing. Copy .env.example to .env and fill it in."
set -a
# shellcheck disable=SC1091
. ./.env
set +a
PORT="${SERVER_PORT:-8080}"
DB="${MYSQL_DATABASE:-chat_db}"
export MYSQL_PWD="${MYSQL_PASSWORD:-}" # keeps the password off the command line
MYSQL=(mysql -h "${MYSQL_HOST:-localhost}" -P "${MYSQL_PORT:-3306}" -u "${MYSQL_USER:-root}")

# 2. MySQL is running.
if ! "${MYSQL[@]}" -e "SELECT 1" >/dev/null 2>&1; then
  if command -v brew >/dev/null && brew services list 2>/dev/null | grep -q '^mysql'; then
    say "Starting MySQL (brew services)..."
    brew services start mysql >/dev/null
    for _ in $(seq 1 30); do
      "${MYSQL[@]}" -e "SELECT 1" >/dev/null 2>&1 && break
      sleep 1
    done
  fi
  "${MYSQL[@]}" -e "SELECT 1" >/dev/null 2>&1 || fail "cannot reach MySQL as ${MYSQL_USER:-root}. Is it running, and are the MYSQL_* values in .env right?"
fi

# 3. The schema is in place.
if ! "${MYSQL[@]}" "$DB" -e "SELECT 1 FROM users LIMIT 1" >/dev/null 2>&1; then
  fail "database '$DB' has no tables yet. Run: make db-create migrate"
fi

# 4. Optional extras.
if [ -n "${GEOIP_DB_PATH:-}" ] && [ ! -f "$GEOIP_DB_PATH" ]; then
  say "note: $GEOIP_DB_PATH not found, so country flags are off. Run: make geoip"
fi

# 5. The port is free.
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
  fail "port $PORT is already in use (the app may already be running). Stop it, or open http://localhost:$PORT"
fi

# 6. Build and run.
say "Building..."
go build -o bin/server ./cmd/server
say "Running on http://localhost:$PORT  (faces: http://localhost:$PORT/faces). Press Ctrl+C to stop."
exec ./bin/server
