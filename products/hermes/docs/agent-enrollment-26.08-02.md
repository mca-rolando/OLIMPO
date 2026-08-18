# HermesDDNS 26.08-02 — Secure Agent Enrollment

This block implements the one-time bootstrap sequence from the HermesDDNS architecture specification. It replaces manual copying of a permanent `hagent_...` credential with a short-lived, single-use enrollment authorization.

## Credential separation

HermesDDNS now uses three deliberately separate secret namespaces:

```text
hddns_...    DDNS / inadyn authentication to /nic/update
hagent_...   permanent Hermes UDM Agent identity
henroll_...  short-lived, one-time Agent bootstrap authorization
```

An enrollment token is never accepted as a DDNS credential or permanent Agent identity. A Device identity credential is never accepted by the DDNS endpoint.

## Enrollment states

```text
pending -> issued -> completed
   |          |
   |          +-> revoked
   +-> expired
   +-> revoked
```

- `pending`: enrollment token exists and may be exchanged once before `expires_at`.
- `issued`: the token has already been consumed and a permanent `hagent_...` Device identity has been issued. The enrollment token can no longer be replayed.
- `completed`: the Agent authenticated with the issued `hagent_...` credential and confirmed that it persisted the identity.
- `revoked`: an administrator invalidated the enrollment. If a permanent identity had already been issued but not confirmed, that identity is revoked in the same transaction.
- `expired`: the `pending` token lifetime elapsed before exchange.

The default enrollment lifetime is 15 minutes. The API accepts 1 through 1440 minutes.

## Security properties

- Enrollment tokens use `henroll_<id>.<secret>` with 256 bits of random secret material.
- Hermes returns the plaintext enrollment token only when the administrator creates it.
- The database stores only the SHA-256 hash of the complete token.
- The token ID is used for indexed lookup before constant-time hash verification.
- Token exchange is single-use. Hermes atomically claims the `pending` enrollment before creating the permanent Agent identity inside the same database transaction.
- A failed transaction rolls the claim back; a successful transaction leaves the enrollment in `issued` and prevents replay.
- Expired, revoked, issued, or completed tokens cannot be exchanged again.
- A Device may have only one open enrollment and only one active permanent Agent identity in this milestone.
- The enrollment token itself determines the Device. `/api/v1/enroll` accepts no caller-supplied Device ID.
- Secret hashes are excluded from API JSON responses.
- Revoking an `issued` but unconfirmed enrollment also revokes the `hagent_...` identity it created.
- Enrollment confirmation is idempotent so the Agent can safely retry after a lost HTTP response.

## Administrative API

### Create enrollment

`POST /api/v1/devices/:id/enrollments`

Optional request:

```json
{
  "ttl_minutes": 15
}
```

The response returns `enrollment_token` exactly once. Example shape:

```json
{
  "enrollment": {
    "device_id": 12,
    "token_id": "0123456789abcdef",
    "status": "pending",
    "expires_at": "2026-08-13T17:15:00Z"
  },
  "enrollment_token": "henroll_0123456789abcdef.<secret>",
  "warning": "Enrollment token is short-lived, single-use, and returned once. Hermes stores only its hash."
}
```

Creating another enrollment while one is `pending` or `issued` returns a conflict. An active permanent Agent identity also prevents creating a new enrollment; revoke the old identity or unfinished enrollment first.

### List enrollments

`GET /api/v1/devices/:id/enrollments`

Pending tokens whose deadline has elapsed are reconciled to `expired` before the list is returned.

### Get enrollment

`GET /api/v1/devices/:id/enrollments/:enrollment_id`

The Device ID is part of the lookup, so an enrollment belonging to another Device is not returned.

### Revoke enrollment

`POST /api/v1/devices/:id/enrollments/:enrollment_id/revoke`

For a `pending` enrollment, this invalidates only the one-time token. For an `issued` enrollment, Hermes also revokes the associated permanent Agent identity so an interrupted bootstrap cannot leave an unconfirmed credential active.

A completed enrollment is historical evidence of a successful bootstrap; revoke its Agent identity through the Agent credential administration endpoint instead.

## Bootstrap exchange

`POST /api/v1/enroll`

Authentication:

```text
Authorization: Bearer henroll_<id>.<secret>
```

This endpoint is intentionally outside the permanent Agent-authenticated route group because a new Agent does not yet possess an `hagent_...` identity.

A successful exchange returns:

- the Device metadata bound to the enrollment;
- the enrollment in `issued` state;
- the newly created Device identity credential metadata;
- the permanent `hagent_...` key exactly once;
- non-secret DDNS bootstrap configuration: Device username, `/nic/update`, and allowed domains;
- the path the Agent must call after persisting its permanent identity.

The endpoint does **not** return a plaintext `hddns_...` credential. Existing DDNS provisioning remains unchanged in this block; automated DDNS configuration delivery is a later Agent milestone.

## Confirmation

After securely persisting the returned `hagent_...` identity, the Agent calls:

`POST /api/v1/agent/enrollment/confirm`

Authentication:

```text
Authorization: Bearer hagent_<id>.<secret>
```

Optional request:

```json
{
  "agent_version": "26.08-02"
}
```

Hermes derives the Device and Agent credential exclusively from the authenticated identity, finds the matching `issued` enrollment, changes it to `completed`, and records Device presence information (`last_seen_at`, `last_ip`, and an optional Agent version).

Repeating the same confirmation with the same permanent identity is safe and returns success.

## End-to-end sequence

```mermaid
sequenceDiagram
  participant Admin
  participant Hermes
  participant Agent

  Admin->>Hermes: POST /devices/:id/enrollments
  Hermes-->>Admin: henroll_... (returned once)
  Admin->>Agent: Official install command + enrollment token
  Agent->>Hermes: POST /enroll, Bearer henroll_...
  Hermes->>Hermes: Validate hash + expiry + single-use state
  Hermes->>Hermes: Atomically claim token
  Hermes->>Hermes: Issue hagent_ identity; store hash only
  Hermes-->>Agent: hagent_... once + Device + non-secret DDNS config
  Agent->>Agent: Persist hagent_ securely
  Agent->>Hermes: POST /agent/enrollment/confirm, Bearer hagent_...
  Hermes->>Hermes: Enrollment = completed; update Device presence
```

## Compatibility and next milestone

The manual administrative Agent credential endpoints remain available for controlled recovery/testing. New installations should use one-time enrollment.

This block establishes secure identity bootstrap only. Installation persistence on UniFi OS, heartbeat/reporting, managed `inadyn` configuration, and Agent software update delivery remain subsequent milestones.
