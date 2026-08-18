#!/usr/bin/env bash
set -euo pipefail
BASE_URL="${HERMES_URL:-http://127.0.0.1:8080}"
ADMIN_USER="${HERMES_ADMIN_USER:-admin}"
ADMIN_PASS="${HERMES_ADMIN_PASS:-}"
DOMAIN="${HERMES_TEST_DOMAIN:-ddns.example.com}"
DEVICE="${HERMES_TEST_DEVICE:-COR-P-TEST}"

curl -fsS "$BASE_URL/health"; echo
AUTH=()
if [[ -n "$ADMIN_PASS" ]]; then AUTH=(-u "$ADMIN_USER:$ADMIN_PASS"); fi
curl -fsS "${AUTH[@]}" -H 'Content-Type: application/json' -d "{\"name\":\"$DOMAIN\",\"default_ttl\":300}" "$BASE_URL/api/v1/domains" || true
RESP=$(curl -fsS "${AUTH[@]}" -H 'Content-Type: application/json' -d "{\"name\":\"$DEVICE\",\"type\":\"UDM\"}" "$BASE_URL/api/v1/devices")
echo "$RESP"
echo "Use the returned api_key with:"
echo "curl -u '$DEVICE:<APIKEY>' '$BASE_URL/nic/update?hostname=${DEVICE,,}.$DOMAIN&myip=203.0.113.10'"
