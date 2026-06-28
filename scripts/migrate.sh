#!/usr/bin/env bash
# Apply or rollback SQL migrations with golang-migrate (PR03+).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT/migrations}"
DATABASE_URL="${ET_MIGRATE_DATABASE_URL:-${ET_DATABASE_URL:-${DATABASE_URL:-}}}"

if [[ -z "${DATABASE_URL}" ]]; then
  echo "ET_MIGRATE_DATABASE_URL, ET_DATABASE_URL, or DATABASE_URL is required" >&2
  exit 1
fi

if ! command -v migrate >/dev/null 2>&1; then
  echo "golang-migrate CLI not found. Install:" >&2
  echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3" >&2
  echo "Or use: docker run --rm -v \"\$PWD/migrations:/migrations\" migrate/migrate ..." >&2
  exit 1
fi

CMD="${1:-up}"
shift || true

case "$CMD" in
  up|down|drop|version|force|goto)
    exec migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" "$CMD" "$@"
    ;;
  *)
    echo "usage: $0 <up|down|drop|version|force|goto> [args...]" >&2
    exit 2
    ;;
esac
