#!/usr/bin/env bash
# Smoke restore drill against a disposable DB URL (set ET_DATABASE_URL).
set -euo pipefail
TMP=$(mktemp)
./scripts/backup-pg.sh "$TMP"
echo "backup ok: $TMP (restore with restore-pg.sh on a scratch DB)"
rm -f "$TMP"
