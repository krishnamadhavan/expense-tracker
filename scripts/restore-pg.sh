#!/usr/bin/env bash
set -euo pipefail
IN="${1:?sql file}"
URL="${ET_DATABASE_URL:?ET_DATABASE_URL required}"
docker run --rm -i --network host postgres:16-alpine psql "$URL" < "$IN"
