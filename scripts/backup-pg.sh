#!/usr/bin/env bash
set -euo pipefail
OUT="${1:-backup-$(date +%Y%m%d).sql}"
URL="${ET_DATABASE_URL:?ET_DATABASE_URL required}"
docker run --rm --network host postgres:16-alpine pg_dump "$URL" > "$OUT"
echo "wrote $OUT"
