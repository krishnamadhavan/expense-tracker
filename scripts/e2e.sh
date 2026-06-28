#!/usr/bin/env bash
set -euo pipefail
BASE="${BASE_URL:-http://127.0.0.1:8080}"
jar=$(mktemp)
csrf=$(curl -s -c "$jar" -b "$jar" -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@localhost","password":"changeme"}' | sed -n 's/.*"csrf_token":"\([^"]*\)".*/\1/p')
test -n "$csrf"
acc=$(curl -s -b "$jar" "$BASE/api/v1/accounts" | sed -n 's/.*"ID":"\([^"]*\)".*/\1/p' | head -1)
# Go json uses capital ID from domain structs - also try lowercase
if [ -z "$acc" ]; then acc=$(curl -s -b "$jar" "$BASE/api/v1/accounts" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['items'][0].get('ID') or d['items'][0].get('id'))"); fi
curl -s -b "$jar" -X POST "$BASE/api/v1/transactions" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $csrf" -H "Idempotency-Key: e2e-1" \
  -d "{\"account_id\":\"$acc\",\"direction\":\"expense\",\"amount\":\"1.00\",\"currency\":\"INR\",\"txn_date\":\"2025-06-01\",\"payee_raw\":\"e2e\"}" | grep -q transaction
echo e2e_ok
