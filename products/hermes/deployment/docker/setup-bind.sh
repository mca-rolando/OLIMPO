#!/bin/sh
set -eu
: "${HERMES_DOMAINS:?HERMES_DOMAINS is required}"
: "${HERMES_PARENT_NS:?HERMES_PARENT_NS is required}"
TTL="${HERMES_DEFAULT_TTL:-300}"
PUBLIC_IP="${HERMES_PUBLIC_IP:-}"
if [ -z "$PUBLIC_IP" ]; then
  PUBLIC_IP="$(curl -4 -fsS --max-time 10 https://icanhazip.com 2>/dev/null | tr -d '\r\n' || true)"
fi
[ -n "$PUBLIC_IP" ] || PUBLIC_IP="127.0.0.1"

for zone in $(echo "$HERMES_DOMAINS" | tr ',' ' '); do
  zone=$(echo "$zone" | sed 's/[[:space:]]//g; s/\.$//')
  [ -n "$zone" ] || continue
  if ! grep -q "zone \"$zone\"" /etc/bind/named.conf.local 2>/dev/null; then
    cat >> /etc/bind/named.conf.local <<ZONE
zone "$zone" {
  type master;
  file "/var/cache/bind/$zone.zone";
  allow-query { any; };
  allow-transfer { none; };
  allow-update { 127.0.0.1; ::1; };
};
ZONE
  fi
  if [ ! -f "/var/cache/bind/$zone.zone" ]; then
    serial=$(date +%Y%m%d%H)
    cat > "/var/cache/bind/$zone.zone" <<ZONE
\$ORIGIN .
\$TTL 86400
$zone. IN SOA ${HERMES_PARENT_NS}. hostmaster.$zone. (
  $serial 3600 900 604800 300 )
  IN NS ${HERMES_PARENT_NS}.
\$ORIGIN $zone.
\$TTL $TTL
@ IN A $PUBLIC_IP
ZONE
  fi
done
chown -R bind:bind /var/cache/bind
chmod 0750 /var/cache/bind
find /var/cache/bind -type f -exec chmod 0640 {} \;
named-checkconf
for zone in $(echo "$HERMES_DOMAINS" | tr ',' ' '); do
  zone=$(echo "$zone" | sed 's/[[:space:]]//g; s/\.$//')
  named-checkzone "$zone" "/var/cache/bind/$zone.zone"
done
