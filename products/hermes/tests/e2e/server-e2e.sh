#!/usr/bin/env bash
set -euo pipefail

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$E2E_DIR/common.sh"

require_commands docker curl python3 git

WORK="$(mktemp -d "${TMPDIR:-/tmp}/hermes-e2e-server.XXXXXX")"
DATA_VOLUME="hermes-e2e-server-data-$$"
BIND_VOLUME="hermes-e2e-server-bind-$$"
BACKUP_VOLUME="hermes-e2e-server-backups-$$"
CONTAINER="hermes-e2e-server-$$"
PORT="$(pick_port)"
BASE_URL="http://127.0.0.1:$PORT"
DOMAIN="e2e.hermes.test"
DEVICE="E2E-UDM-01"
FQDN="${DEVICE,,}.$DOMAIN"

cleanup() {
  local rc=$?
  trap - EXIT
  if ((rc != 0)); then
    printf '\n[hermes-e2e] container logs after failure:\n' >&2
    docker logs "$CONTAINER" >&2 2>/dev/null || true
  fi
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  docker volume rm -f "$DATA_VOLUME" "$BIND_VOLUME" "$BACKUP_VOLUME" >/dev/null 2>&1 || true
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

IMAGE="$(ensure_current_image)"
log "starting $IMAGE on $BASE_URL without publishing host DNS port 53"
run_hermes_container \
  "$IMAGE" "$CONTAINER" "$PORT" \
  "$DATA_VOLUME" "$BIND_VOLUME" "$BACKUP_VOLUME" "$DOMAIN"
wait_health "$BASE_URL" "$CONTAINER"
wait_dns "$CONTAINER" "$DOMAIN"
log "health and authoritative BIND checks passed"

DEVICE_JSON="$(create_device "$BASE_URL" "$DEVICE")"
DEVICE_ID="$(printf '%s' "$DEVICE_JSON" | json_get 'device.ID')"
DDNS_KEY="$(printf '%s' "$DEVICE_JSON" | json_get 'api_key')"
assert_nonempty "$DEVICE_ID" "device id was not returned"
assert_nonempty "$DDNS_KEY" "DDNS key was not returned"
log "device created with id $DEVICE_ID"

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.10')"
assert_eq "$RESP" 'good 192.0.2.10' 'first IPv4 DDNS update'
A_RECORD="$(container_dig "$CONTAINER" "$FQDN" A)"
assert_eq "$A_RECORD" '192.0.2.10' 'BIND A record after first update'

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.10')"
assert_eq "$RESP" 'nochg 192.0.2.10' 'unchanged IPv4 DDNS update'
log "DDNS good/nochg behavior verified"

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '2001:db8::10')"
assert_eq "$RESP" 'good 2001:db8::10' 'IPv4-to-IPv6 DDNS transition'
AAAA_RECORD="$(container_dig "$CONTAINER" "$FQDN" AAAA)"
assert_eq "$AAAA_RECORD" '2001:db8::10' 'BIND AAAA record after transition'
A_RECORD="$(container_dig "$CONTAINER" "$FQDN" A)"
assert_eq "$A_RECORD" '' 'stale A record after IPv4-to-IPv6 transition'

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.11')"
assert_eq "$RESP" 'good 192.0.2.11' 'IPv6-to-IPv4 DDNS transition'
A_RECORD="$(container_dig "$CONTAINER" "$FQDN" A)"
assert_eq "$A_RECORD" '192.0.2.11' 'BIND A record after reverse transition'
AAAA_RECORD="$(container_dig "$CONTAINER" "$FQDN" AAAA)"
assert_eq "$AAAA_RECORD" '' 'stale AAAA record after IPv6-to-IPv4 transition'
log "A/AAAA stale-record protection verified against live BIND"

ENROLLMENT_JSON="$(create_agent_enrollment "$BASE_URL" "$DEVICE_ID")"
ENROLLMENT_TOKEN="$(printf '%s' "$ENROLLMENT_JSON" | json_get 'enrollment_token')"
assert_nonempty "$ENROLLMENT_TOKEN" "enrollment token was not returned"

EXCHANGE_JSON="$(exchange_agent_enrollment "$BASE_URL" "$ENROLLMENT_TOKEN")"
AGENT_KEY="$(printf '%s' "$EXCHANGE_JSON" | json_get 'agent_key')"
assert_nonempty "$AGENT_KEY" "agent identity key was not returned"

REUSE_STATUS="$(curl -sS -o "$WORK/reused-enrollment.json" -w '%{http_code}' \
  -X POST -H "Authorization: Bearer $ENROLLMENT_TOKEN" "$BASE_URL/api/v1/enroll")"
assert_eq "$REUSE_STATUS" '401' 'single-use enrollment token reuse'

CONFIRM_JSON="$(confirm_agent_enrollment "$BASE_URL" "$AGENT_KEY")"
CONFIRM_STATUS="$(printf '%s' "$CONFIRM_JSON" | json_get 'enrollment.status')"
assert_eq "$CONFIRM_STATUS" 'completed' 'agent enrollment confirmation'

ME_JSON="$(curl -fsS -H "Authorization: Bearer $AGENT_KEY" "$BASE_URL/api/v1/agent/me")"
ME_DEVICE_ID="$(printf '%s' "$ME_JSON" | json_get 'device.ID')"
assert_eq "$ME_DEVICE_ID" "$DEVICE_ID" 'agent identity device binding'
log "secure one-time enrollment and permanent Agent identity verified"

HEARTBEAT_JSON="$(curl -fsS \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"agent_version":"26.08-02-e2e","system_hostname":"e2e-udm-01","platform":"unifi-os","architecture":"arm64","firmware_version":"e2e","uptime_seconds":3600,"cpu_count":4,"load_1":0.25,"memory_total_bytes":4294967296,"memory_available_bytes":2147483648,"disk_total_bytes":17179869184,"disk_available_bytes":8589934592}' \
  "$BASE_URL/api/v1/agent/heartbeat")"
HB_STATUS="$(printf '%s' "$HEARTBEAT_JSON" | json_get 'status')"
assert_eq "$HB_STATUS" 'ok' 'Agent heartbeat response'

STATUS_JSON="$(curl -fsS "$BASE_URL/api/v1/devices/$DEVICE_ID/agent-status")"
AGENT_STATE="$(printf '%s' "$STATUS_JSON" | json_get 'state')"
AGENT_VERSION="$(printf '%s' "$STATUS_JSON" | json_get 'telemetry.agent_version')"
assert_eq "$AGENT_STATE" 'online' 'Agent online state after heartbeat'
assert_eq "$AGENT_VERSION" '26.08-02-e2e' 'Agent telemetry version'
log "heartbeat/current telemetry verified"

NETWORK_JSON="$(curl -fsS \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"wans":[{"interface_name":"eth8","role":"primary","default_route":true,"ipv4":"10.10.10.2","gateway_ipv4":"10.10.10.1","public_ipv4":"8.8.8.8"}],"networks":[{"name":"LAN","vlan_id":10,"ipv4_cidr":"10.20.30.0/24","gateway_ipv4":"10.20.30.1","purpose":"corporate"}]}' \
  "$BASE_URL/api/v1/agent/network-context")"
NAT_STATE="$(printf '%s' "$NETWORK_JSON" | json_get 'wans.0.nat_state')"
PUBLIC_SOURCE="$(printf '%s' "$NETWORK_JSON" | json_get 'wans.0.public_ip_source')"
assert_eq "$NAT_STATE" 'double_nat' 'private WAN/public probe NAT classification'
assert_eq "$PUBLIC_SOURCE" 'agent_probe' 'public IP source classification'

MULTI_DEFAULT_STATUS="$(curl -sS -o "$WORK/multiple-default-routes.json" -w '%{http_code}' \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"wans":[{"interface_name":"eth8","role":"primary","default_route":true,"ipv4":"10.10.10.2"},{"interface_name":"eth9","role":"secondary","default_route":true,"ipv4":"10.10.20.2"}],"networks":[]}' \
  "$BASE_URL/api/v1/agent/network-context")"
assert_eq "$MULTI_DEFAULT_STATUS" '400' 'multiple default WAN routes must be rejected'
log "network identity snapshot and default-route validation verified"

ROTATION_JSON="$(curl -fsS \
  -H 'Content-Type: application/json' \
  -d '{"grace_minutes":1}' \
  "$BASE_URL/api/v1/devices/$DEVICE_ID/credential-rotations")"
ROTATION_ID="$(printf '%s' "$ROTATION_JSON" | json_get 'rotation.ID')"
assert_nonempty "$ROTATION_ID" "rotation id was not returned"

CURRENT_JSON="$(curl -fsS -H "Authorization: Bearer $AGENT_KEY" \
  "$BASE_URL/api/v1/agent/credential-rotations/current")"
CURRENT_STATUS="$(printf '%s' "$CURRENT_JSON" | json_get 'rotation.status')"
assert_eq "$CURRENT_STATUS" 'requested' 'Agent current rotation after request'

read -r CANDIDATE_ID CANDIDATE_KEY CANDIDATE_HASH < <(python3 - <<'PY'
import base64
import hashlib
import secrets
kid = secrets.token_hex(8)
secret = base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode()
key = f"hddns_{kid}.{secret}"
print(kid, key, hashlib.sha256(key.encode()).hexdigest())
PY
)

STAGE_JSON="$(curl -fsS \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"key_id\":\"$CANDIDATE_ID\",\"secret_hash\":\"$CANDIDATE_HASH\"}" \
  "$BASE_URL/api/v1/agent/credential-rotations/$ROTATION_ID/stage")"
STAGED_STATUS="$(printf '%s' "$STAGE_JSON" | json_get 'credential.status')"
assert_eq "$STAGED_STATUS" 'pending' 'staged DDNS candidate state'

VALIDATION_JSON="$(curl -fsS -X POST \
  -H "Authorization: Bearer $AGENT_KEY" \
  "$BASE_URL/api/v1/agent/credential-rotations/$ROTATION_ID/start-validation")"
VALIDATION_STATUS="$(printf '%s' "$VALIDATION_JSON" | json_get 'rotation.status')"
assert_eq "$VALIDATION_STATUS" 'validating' 'credential rotation validation state'

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$CANDIDATE_KEY" "$FQDN" '192.0.2.12')"
assert_eq "$RESP" 'good 192.0.2.12' 'DDNS update with staged candidate'

ROTATION_STATE_JSON="$(curl -fsS "$BASE_URL/api/v1/devices/$DEVICE_ID/credential-rotations/$ROTATION_ID")"
ROTATION_STATUS="$(printf '%s' "$ROTATION_STATE_JSON" | json_get 'status')"
assert_eq "$ROTATION_STATUS" 'grace' 'rotation state after candidate DDNS confirmation'

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.12')"
assert_eq "$RESP" 'nochg 192.0.2.12' 'previous DDNS credential during grace'
RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$CANDIDATE_KEY" "$FQDN" '192.0.2.12')"
assert_eq "$RESP" 'nochg 192.0.2.12' 'replacement DDNS credential during grace'
log "Agent-driven DDNS credential rotation reached grace successfully"

LOGS_JSON="$(curl -fsS "$BASE_URL/api/v1/logs")"
LOG_COUNT="$(printf '%s' "$LOGS_JSON" | json_count)"
if ((LOG_COUNT < 6)); then
  fail "expected DDNS audit log entries, found $LOG_COUNT"
fi

log "PASS: live server + SQLite + BIND + Agent lifecycle E2E completed"
