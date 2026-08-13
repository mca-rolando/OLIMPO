# HermesDDNS 26.08-02 — Agent Heartbeat and Operational Telemetry

This block gives HermesDDNS an Agent-specific presence signal and a bounded current-state telemetry snapshot suitable for the fleet dashboard. It deliberately does **not** implement historical time-series monitoring, VPN discovery, interface/network configuration inventory, or alerting yet.

## Design goals

- Prove that the Hermes UDM Agent itself is alive, independently from `inadyn` DDNS traffic.
- Authenticate every heartbeat with the permanent `hagent_...` Device identity.
- Derive the Device exclusively from the authenticated Agent identity; the heartbeat payload contains no authoritative Device selector.
- Keep one current telemetry row per Device instead of creating an unbounded heartbeat history table.
- Support presence-only heartbeats without erasing previously reported telemetry.
- Provide a fleet-oriented status API that avoids one HTTP request/query per Device.
- Keep the existing `Device.last_seen_at` useful as general Device activity while using `AgentTelemetrySnapshot.last_heartbeat_at` as the authoritative Agent online/offline signal.

## Agent endpoint

`POST /api/v1/agent/heartbeat`

Authentication:

```text
Authorization: Bearer hagent_<id>.<secret>
```

A minimal presence-only heartbeat is valid:

```json
{}
```

A full example is:

```json
{
  "agent_version": "26.08-02",
  "system_hostname": "COR-P-MCAOffice",
  "platform": "unifi-os",
  "architecture": "arm64",
  "os_version": "UniFi OS",
  "kernel_version": "6.x",
  "firmware_version": "4.x",
  "boot_id": "example-boot-id",
  "uptime_seconds": 123456,
  "cpu_count": 4,
  "load_1": 0.21,
  "load_5": 0.18,
  "load_15": 0.16,
  "memory_total_bytes": 4294967296,
  "memory_available_bytes": 2147483648,
  "disk_total_bytes": 137438953472,
  "disk_available_bytes": 107374182400
}
```

Fields are optional. Omitted fields retain the last reported value; this lets the Agent send a lightweight heartbeat if collection of one telemetry source temporarily fails.

Hermes records the heartbeat time using **server time**, not a Device-provided timestamp. The caller IP is also derived server-side using the existing trusted-proxy policy.

Successful response:

```json
{
  "status": "ok",
  "device_id": 1,
  "server_time": "2026-08-13T17:00:00Z",
  "next_heartbeat_seconds": 60,
  "online_threshold_seconds": 180,
  "telemetry": {}
}
```

The interval and threshold are protocol defaults for this milestone. They can become server configuration later without changing the Agent identity model.

## Agent state semantics

Hermes exposes three Agent states:

```text
never_seen  no Agent heartbeat has ever been received
online      last Agent heartbeat is <= 180 seconds old
offline     an Agent heartbeat exists but is > 180 seconds old
```

This state intentionally does not use `Device.last_seen_at`, because DDNS updates also refresh that generic field. A functioning `inadyn` process must not make a failed Hermes Agent appear online.

## Administrative status API

Per Device:

`GET /api/v1/devices/:id/agent-status`

Fleet view:

`GET /api/v1/agent-status`

Each status object includes:

- the Device metadata;
- `state` and boolean `online`;
- heartbeat age in seconds when a heartbeat exists;
- the current `AgentTelemetrySnapshot`, or `null` for `never_seen`.

The fleet endpoint loads Devices and telemetry snapshots in two bounded queries and joins them in memory. It is intended as the backend primitive for the future 1080p dashboard rather than forcing a browser to issue roughly one request per UDM.

## Storage model

`AgentTelemetrySnapshot` is a one-to-zero-or-one relationship with `Device`.

The row contains only the latest known telemetry and heartbeat metadata. A heartbeat updates the same row. This prevents a 60-second heartbeat from creating approximately 1,440 rows per Device per day.

Historical measurements, retention policy, aggregation, metrics export, and alert history belong in a later monitoring milestone if operational requirements justify them.

## Validation and security

- Agent authentication uses the existing `hagent_...` Bearer middleware.
- A heartbeat cannot select another Device by URL or request field; the Device comes from the authenticated credential.
- Revoked/expired Agent identity credentials therefore cannot heartbeat.
- `memory_available_bytes` cannot exceed `memory_total_bytes` when total memory is known.
- `disk_available_bytes` cannot exceed `disk_total_bytes` when total disk size is known.
- CPU count and load values have defensive bounds.
- No Agent/DDNS/enrollment plaintext secret or stored hash is returned by heartbeat/status APIs.

## Scope boundary

This block is the Agent presence/operational-health foundation only. The following remain separate subsequent milestones:

- UDM-side persistent Agent executable/service;
- managed `inadyn` configuration delivery;
- interface and VLAN inventory;
- WAN state and public-IP telemetry;
- VPN tunnel discovery/status;
- detailed UniFi network configuration summary;
- alerting/escalation;
- historical telemetry/time-series retention;
- Agent release/update delivery.
