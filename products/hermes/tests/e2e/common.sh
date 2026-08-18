#!/usr/bin/env bash
set -euo pipefail

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$E2E_DIR/../.." && pwd)"

log() {
  printf '[hermes-e2e] %s\n' "$*"
}

fail() {
  printf '[hermes-e2e] ERROR: %s\n' "$*" >&2
  return 1
}

require_commands() {
  local cmd
  for cmd in "$@"; do
    command -v "$cmd" >/dev/null 2>&1 || fail "required command not found: $cmd"
  done
}

pick_port() {
  python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

json_get() {
  local path="$1"
  python3 -c '
import json
import sys

def key_for(obj, wanted):
    if wanted in obj:
        return wanted
    low = wanted.lower()
    for key in obj:
        if isinstance(key, str) and key.lower() == low:
            return key
    raise KeyError(wanted)

data = json.load(sys.stdin)
for part in sys.argv[1].split("."):
    if isinstance(data, list):
        data = data[int(part)]
    elif isinstance(data, dict):
        data = data[key_for(data, part)]
    else:
        raise KeyError(part)

if isinstance(data, bool):
    print("true" if data else "false")
elif data is None:
    print("null")
elif isinstance(data, (dict, list)):
    print(json.dumps(data, separators=(",", ":")))
else:
    print(data)
' "$path"
}

json_count() {
  python3 -c 'import json,sys; print(len(json.load(sys.stdin)))'
}

assert_eq() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [[ "$actual" != "$expected" ]]; then
    fail "$message: expected '$expected', got '$actual'"
  fi
}

assert_nonempty() {
  local value="$1"
  local message="$2"
  [[ -n "$value" ]] || fail "$message"
}

wait_health() {
  local base_url="$1"
  local container_name="$2"
  local attempts="${3:-120}"
  local i
  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS "$base_url/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  docker logs "$container_name" >&2 || true
  fail "HermesDDNS did not become healthy at $base_url"
}

wait_dns() {
  local container_name="$1"
  local domain="$2"
  local attempts="${3:-60}"
  local i
  for ((i = 1; i <= attempts; i++)); do
    if docker exec "$container_name" dig +time=1 +tries=1 +short @127.0.0.1 "$domain" SOA 2>/dev/null | grep -q .; then
      return 0
    fi
    sleep 0.5
  done
  docker logs "$container_name" >&2 || true
  fail "BIND did not become authoritative for $domain"
}

current_image_name() {
  local commit
  commit="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf 'working')"
  if [[ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null || true)" ]]; then
    printf 'hermesddns-e2e:%s-working\n' "$commit"
    return
  fi
  printf 'hermesddns-e2e:%s\n' "$commit"
}

ensure_current_image() {
  local image commit dirty
  image="$(current_image_name)"
  commit="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf 'working')"
  dirty=false
  if [[ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null || true)" ]]; then
    dirty=true
  fi

  if [[ "$dirty" == true ]] || ! docker image inspect "$image" >/dev/null 2>&1; then
    log "building current HermesDDNS image $image" >&2
    if ! docker build \
      -f "$ROOT/deployment/docker/Dockerfile" \
      --build-arg VERSION=26.08-02-e2e \
      --build-arg COMMIT="$commit" \
      --build-arg BUILD_TIME=e2e \
      -t "$image" \
      "$ROOT" >&2; then
      fail "failed to build current HermesDDNS E2E image $image" >&2
      return 1
    fi
  fi
  printf '%s\n' "$image"
}

run_hermes_container() {
  local image="$1"
  local container_name="$2"
  local port="$3"
  local data_volume="$4"
  local bind_volume="$5"
  local backup_volume="$6"
  local domain="$7"

  docker volume create "$data_volume" >/dev/null
  docker volume create "$bind_volume" >/dev/null
  docker volume create "$backup_volume" >/dev/null

  docker run -d \
    --name "$container_name" \
    -p "127.0.0.1:${port}:8080" \
    -e HERMES_LISTEN=:8080 \
    -e HERMES_DATABASE=/opt/hermesddns/data/hermes.db \
    -e HERMES_DOMAINS="$domain" \
    -e HERMES_PARENT_NS=ns.hermes.test \
    -e HERMES_DEFAULT_TTL=60 \
    -e HERMES_ALLOW_WILDCARD=false \
    -e HERMES_DNS_SERVER=127.0.0.1 \
    -e HERMES_NSUPDATE_BINARY=/usr/bin/nsupdate \
    -e HERMES_TRUST_PROXY_HEADERS=false \
    -e HERMES_AUTOCREATE_POLICY=device-prefix \
    -e HERMES_ALLOW_INSECURE_ADMIN=true \
    -e HERMES_PUBLIC_IP=127.0.0.1 \
    -v "$data_volume:/opt/hermesddns/data" \
    -v "$bind_volume:/var/cache/bind" \
    -v "$backup_volume:/opt/hermesddns/backups" \
    "$image" >/dev/null
}

container_dig() {
  local container_name="$1"
  local fqdn="$2"
  local record_type="$3"
  docker exec "$container_name" dig +short @127.0.0.1 "$fqdn" "$record_type" | tr -d '\r' | sed '/^[[:space:]]*$/d'
}

create_device() {
  local base_url="$1"
  local name="$2"
  local type="${3:-UDM-SE}"
  curl -fsS \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$name\",\"display_name\":\"Hermes E2E\",\"type\":\"$type\"}" \
    "$base_url/api/v1/devices"
}

perform_ddns_update() {
  local base_url="$1"
  local username="$2"
  local api_key="$3"
  local fqdn="$4"
  local ip="$5"
  curl -fsS \
    --user "$username:$api_key" \
    --get \
    --data-urlencode "hostname=$fqdn" \
    --data-urlencode "myip=$ip" \
    "$base_url/nic/update" | tr -d '\r\n'
}

create_agent_enrollment() {
  local base_url="$1"
  local device_id="$2"
  curl -fsS \
    -H 'Content-Type: application/json' \
    -d '{"ttl_minutes":15}' \
    "$base_url/api/v1/devices/$device_id/enrollments"
}

exchange_agent_enrollment() {
  local base_url="$1"
  local token="$2"
  curl -fsS \
    -X POST \
    -H "Authorization: Bearer $token" \
    "$base_url/api/v1/enroll"
}

confirm_agent_enrollment() {
  local base_url="$1"
  local agent_key="$2"
  local version="${3:-26.08-02-e2e}"
  curl -fsS \
    -H "Authorization: Bearer $agent_key" \
    -H 'Content-Type: application/json' \
    -d "{\"agent_version\":\"$version\"}" \
    "$base_url/api/v1/agent/enrollment/confirm"
}
