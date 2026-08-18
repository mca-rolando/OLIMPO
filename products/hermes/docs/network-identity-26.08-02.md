# HermesDDNS 26.08-02 — UDM Network Identity Snapshot

This block adds a deliberately small, current-state network identity inventory for each managed UDM. It gives Hermes enough context to understand how a Device reaches the Internet and which routed/VLAN networks exist behind it without turning Hermes into a network-monitoring platform.

VPN health, latency, packet loss, switches, access points, clients, cameras, traffic analytics, incidents, SLA history, and topology monitoring are intentionally outside this contract. Those responsibilities belong to ARGUS.

## Scope

Hermes stores only the latest complete network identity snapshot reported by the authenticated Agent:

- WAN interface name.
- WAN role (`primary`, `secondary`, or `other`).
- Whether the interface is the default route used by the report.
- Local WAN IPv4 and optional IPv6.
- WAN IPv4 gateway.
- Public IPv4 observed for that WAN when available.
- Public-address classification: `public`, `private`, `cgnat`, `special`, or `unknown`.
- NAT classification: `direct`, `double_nat`, `cgnat`, `public_mismatch`, or `unknown`.
- LAN/network name.
- VLAN ID when tagged.
- IPv4 subnet/CIDR.
- IPv4 gateway.
- Optional small purpose label.

Hermes stores one `NetworkIdentitySnapshot` per Device and atomically replaces the associated WAN/network rows whenever the Agent submits a new complete snapshot. It does not accumulate a time-series history.

## Agent API

`POST /api/v1/agent/network-context`

Authentication:

```text
Authorization: Bearer hagent_<id>.<secret>
```

The Device ID is derived exclusively from the authenticated Agent identity. A caller-supplied `device_id` is not part of the request contract and cannot redirect the report to another Device.

Example request:

```json
{
  "wans": [
    {
      "interface_name": "eth8",
      "role": "primary",
      "default_route": true,
      "ipv4": "192.168.1.20",
      "gateway_ipv4": "192.168.1.1"
    },
    {
      "interface_name": "eth9",
      "role": "secondary",
      "ipv4": "100.72.35.18",
      "public_ipv4": "8.8.8.8"
    }
  ],
  "networks": [
    {
      "name": "LAN",
      "ipv4_cidr": "10.222.0.0/24",
      "gateway_ipv4": "10.222.0.1",
      "purpose": "corporate"
    },
    {
      "name": "POS",
      "vlan_id": 20,
      "ipv4_cidr": "10.222.20.0/24",
      "gateway_ipv4": "10.222.20.1",
      "purpose": "corporate"
    }
  ]
}
```

At least one WAN interface is required. A report is a complete replacement, not a patch.

## Public/private and NAT detection

Hermes separates the address configured on the UDM WAN from the public address observed outside that interface.

### Address scope

- RFC1918 (`10/8`, `172.16/12`, `192.168/16`) -> `private`.
- Shared-address space `100.64.0.0/10` -> `cgnat`.
- Ordinary globally routable IPv4 -> `public`.
- Loopback, link-local, documentation, benchmark, reserved, and other non-public ranges -> `special`.
- Missing local IPv4 -> `unknown`.

### NAT states

| Local WAN | Observed public IPv4 | Result |
|---|---|---|
| Public and equal | Same address | `direct` |
| RFC1918 private | Public address | `double_nat` |
| `100.64.0.0/10` | Public or unknown | `cgnat` |
| Public | Different public address | `public_mismatch` |
| Insufficient evidence | — | `unknown` |

`public_mismatch` is deliberately not called Double NAT. A different public address can be caused by 1:1 NAT, policy routing, a different egress path, or another upstream design.

### Source of the observed public IPv4

The preferred source is an Agent-side external-IP probe bound to the specific WAN, reported as `public_ipv4`. This is especially important for secondary WANs and installations where Hermes is behind a reverse proxy or tunnel.

When a WAN is marked `default_route`, `public_ipv4` is omitted, and the peer address seen by Hermes is a usable public IPv4, Hermes may use that address as a fallback and marks `public_ip_source` as `server_peer`.

If Hermes is deployed behind a reverse proxy, `HERMES_TRUST_PROXY_HEADERS` must be configured correctly before relying on the server-peer fallback. Agent-probed public IP remains the preferred value.

## Administrative API

### One Device

`GET /api/v1/devices/:id/network-context`

A Device that exists but has never reported network identity returns HTTP 200 with:

```json
{
  "reported": false,
  "snapshot": null,
  "wans": [],
  "networks": []
}
```

### Fleet

`GET /api/v1/network-context`

Returns all Devices ordered by name, including Devices that have not yet reported. The implementation uses bounded fleet queries and groups current WAN/network rows in memory rather than issuing N+1 database queries.

This endpoint is intentionally suitable for a future ARGUS integration to consume Hermes identity data without making ARGUS depend on Hermes internal database tables.

## Storage and validation

New current-state models:

```text
NetworkIdentitySnapshot
NetworkWAN
NetworkSegment
```

Hermes validates:

- non-empty and non-duplicated WAN interface names;
- WAN roles;
- IPv4/IPv6 syntax;
- that an Agent-supplied `public_ipv4` is actually public IPv4;
- VLAN IDs from 1 through 4094;
- IPv4 CIDR syntax;
- that a reported LAN gateway belongs to its reported subnet.

Replacement of WAN/network child rows is transactional and uses hard deletion for the superseded snapshot rows so repeated reports do not create an unbounded soft-delete history.

## Product boundary: Hermes vs. ARGUS

Hermes owns identity-level data:

```text
Device identity
DDNS identity
Agent identity/enrollment
Current public IP context
Basic WAN identity
Public/private/CGNAT/Double-NAT classification
LAN/VLAN/subnet summary
```

ARGUS owns observability:

```text
VPN health
WAN health and performance
Latency / packet loss
UniFi device inventory
Switches / APs / Protect / clients
Traffic and historical metrics
Incidents / alerts / health scoring
NOC dashboards and kiosk wallboards
```

Hermes does not collect VPN state in this block.
