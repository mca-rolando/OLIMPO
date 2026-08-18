#!/usr/bin/env bash
set -euo pipefail

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$E2E_DIR/common.sh"

require_commands docker curl python3 git tar

OLD_REF="${HERMES_UPGRADE_FROM_REF:-26.08-01}"
if ! git -C "$ROOT" rev-parse --verify --quiet "${OLD_REF}^{commit}" >/dev/null; then
  fail "Git ref '$OLD_REF' is required for the upgrade test. Fetch tags/history first."
fi
OLD_COMMIT="$(git -C "$ROOT" rev-parse --short "${OLD_REF}^{commit}")"

WORK="$(mktemp -d "${TMPDIR:-/tmp}/hermes-e2e-upgrade.XXXXXX")"
DATA_VOLUME="hermes-e2e-upgrade-data-$$"
BIND_VOLUME="hermes-e2e-upgrade-bind-$$"
BACKUP_VOLUME="hermes-e2e-upgrade-backups-$$"
OLD_CONTAINER="hermes-e2e-upgrade-old-$$"
NEW_CONTAINER="hermes-e2e-upgrade-new-$$"
PORT="$(pick_port)"
BASE_URL="http://127.0.0.1:$PORT"
DOMAIN="upgrade.hermes.test"
DEVICE="UPGRADE-UDM-01"
FQDN="${DEVICE,,}.$DOMAIN"
OLD_IMAGE="hermesddns-e2e:upgrade-from-$OLD_COMMIT"

cleanup() {
  local rc=$?
  trap - EXIT
  if ((rc != 0)); then
    printf '\n[hermes-e2e] old container logs after failure:\n' >&2
    docker logs "$OLD_CONTAINER" >&2 2>/dev/null || true
    printf '\n[hermes-e2e] new container logs after failure:\n' >&2
    docker logs "$NEW_CONTAINER" >&2 2>/dev/null || true
  fi
  docker rm -f "$OLD_CONTAINER" "$NEW_CONTAINER" >/dev/null 2>&1 || true
  docker volume rm -f "$DATA_VOLUME" "$BIND_VOLUME" "$BACKUP_VOLUME" >/dev/null 2>&1 || true
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

mkdir -p "$WORK/old-src"
git -C "$ROOT" archive --format=tar "$OLD_REF" | tar -xf - -C "$WORK/old-src"

if ! docker image inspect "$OLD_IMAGE" >/dev/null 2>&1; then
  log "building historical HermesDDNS image from $OLD_REF"
  docker build \
    -f "$WORK/old-src/deployment/docker/Dockerfile" \
    --build-arg VERSION=26.08-01 \
    --build-arg COMMIT="$OLD_COMMIT" \
    --build-arg BUILD_TIME=e2e \
    -t "$OLD_IMAGE" \
    "$WORK/old-src"
fi

NEW_IMAGE="$(ensure_current_image)"

log "starting historical $OLD_REF with persistent SQLite/BIND data"
run_hermes_container \
  "$OLD_IMAGE" "$OLD_CONTAINER" "$PORT" \
  "$DATA_VOLUME" "$BIND_VOLUME" "$BACKUP_VOLUME" "$DOMAIN"
wait_health "$BASE_URL" "$OLD_CONTAINER"
wait_dns "$OLD_CONTAINER" "$DOMAIN"

DEVICE_JSON="$(create_device "$BASE_URL" "$DEVICE")"
DEVICE_ID="$(printf '%s' "$DEVICE_JSON" | json_get 'device.ID')"
DDNS_KEY="$(printf '%s' "$DEVICE_JSON" | json_get 'api_key')"
assert_nonempty "$DEVICE_ID" "26.08-01 device id was not returned"
assert_nonempty "$DDNS_KEY" "26.08-01 DDNS key was not returned"

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.21')"
assert_eq "$RESP" 'good 192.0.2.21' '26.08-01 initial DDNS update'
RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.21')"
assert_eq "$RESP" 'nochg 192.0.2.21' '26.08-01 unchanged DDNS update'
OLD_A="$(container_dig "$OLD_CONTAINER" "$FQDN" A)"
assert_eq "$OLD_A" '192.0.2.21' '26.08-01 BIND record'

OLD_LOGS_JSON="$(curl -fsS "$BASE_URL/api/v1/logs")"
OLD_LOG_COUNT="$(printf '%s' "$OLD_LOGS_JSON" | json_count)"
if ((OLD_LOG_COUNT < 2)); then
  fail "26.08-01 did not persist expected audit logs"
fi
log "26.08-01 fixture created: device, credential, host, DNS record and logs"

docker stop -t 5 "$OLD_CONTAINER" >/dev/null

log "starting current 26.08-02 code against the same SQLite/BIND data"
run_hermes_container \
  "$NEW_IMAGE" "$NEW_CONTAINER" "$PORT" \
  "$DATA_VOLUME" "$BIND_VOLUME" "$BACKUP_VOLUME" "$DOMAIN"
wait_health "$BASE_URL" "$NEW_CONTAINER"
wait_dns "$NEW_CONTAINER" "$DOMAIN"

DEVICES_JSON="$(curl -fsS "$BASE_URL/api/v1/devices")"
printf '%s' "$DEVICES_JSON" | python3 -c '
import json
import sys
expected_id = int(sys.argv[1])
expected_name = sys.argv[2]
devices = json.load(sys.stdin)
for device in devices:
    did = device.get("ID", device.get("id"))
    name = device.get("name")
    if did == expected_id and name == expected_name:
        raise SystemExit(0)
raise SystemExit("pre-upgrade device was not preserved")
' "$DEVICE_ID" "$DEVICE"

RESP="$(perform_ddns_update "$BASE_URL" "$DEVICE" "$DDNS_KEY" "$FQDN" '192.0.2.22')"
assert_eq "$RESP" 'good 192.0.2.22' 'pre-upgrade DDNS credential after migration'
NEW_A="$(container_dig "$NEW_CONTAINER" "$FQDN" A)"
assert_eq "$NEW_A" '192.0.2.22' 'BIND record after 26.08-02 startup'

NEW_LOGS_JSON="$(curl -fsS "$BASE_URL/api/v1/logs")"
NEW_LOG_COUNT="$(printf '%s' "$NEW_LOGS_JSON" | json_count)"
if ((NEW_LOG_COUNT <= OLD_LOG_COUNT)); then
  fail "audit log history was not preserved/extended across upgrade"
fi
log "pre-upgrade device, DDNS credential, host, DNS state and logs preserved"

ENROLLMENT_JSON="$(create_agent_enrollment "$BASE_URL" "$DEVICE_ID")"
ENROLLMENT_TOKEN="$(printf '%s' "$ENROLLMENT_JSON" | json_get 'enrollment_token')"
EXCHANGE_JSON="$(exchange_agent_enrollment "$BASE_URL" "$ENROLLMENT_TOKEN")"
AGENT_KEY="$(printf '%s' "$EXCHANGE_JSON" | json_get 'agent_key')"
CONFIRM_JSON="$(confirm_agent_enrollment "$BASE_URL" "$AGENT_KEY" '26.08-02-upgrade-e2e')"
CONFIRM_STATUS="$(printf '%s' "$CONFIRM_JSON" | json_get 'enrollment.status')"
assert_eq "$CONFIRM_STATUS" 'completed' 'post-upgrade enrollment table migration'

HEARTBEAT_JSON="$(curl -fsS \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"agent_version":"26.08-02-upgrade-e2e","system_hostname":"upgrade-udm-01","platform":"unifi-os","uptime_seconds":7200,"cpu_count":4}' \
  "$BASE_URL/api/v1/agent/heartbeat")"
HB_STATUS="$(printf '%s' "$HEARTBEAT_JSON" | json_get 'status')"
assert_eq "$HB_STATUS" 'ok' 'post-upgrade telemetry table migration'

NETWORK_JSON="$(curl -fsS \
  -H "Authorization: Bearer $AGENT_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"wans":[{"interface_name":"eth8","role":"primary","default_route":true,"ipv4":"10.40.50.2","gateway_ipv4":"10.40.50.1","public_ipv4":"8.8.4.4"}],"networks":[{"name":"Corporate","vlan_id":20,"ipv4_cidr":"10.50.60.0/24","gateway_ipv4":"10.50.60.1","purpose":"corporate"}]}' \
  "$BASE_URL/api/v1/agent/network-context")"
REPORTED="$(printf '%s' "$NETWORK_JSON" | json_get 'reported')"
assert_eq "$REPORTED" 'true' 'post-upgrade network identity tables migration'

ROTATION_JSON="$(curl -fsS \
  -H 'Content-Type: application/json' \
  -d '{"grace_minutes":30}' \
  "$BASE_URL/api/v1/devices/$DEVICE_ID/credential-rotations")"
ROTATION_STATUS="$(printf '%s' "$ROTATION_JSON" | json_get 'rotation.status')"
assert_eq "$ROTATION_STATUS" 'requested' 'post-upgrade credential rotation table migration'

STATUS_JSON="$(curl -fsS "$BASE_URL/api/v1/devices/$DEVICE_ID/agent-status")"
AGENT_STATE="$(printf '%s' "$STATUS_JSON" | json_get 'state')"
assert_eq "$AGENT_STATE" 'online' 'post-upgrade Agent status'

log "PASS: real $OLD_REF -> current 26.08-02 persistent-data upgrade completed"
