# HermesDDNS 26.08-02 — Agent Identity and Rotation API

This block introduces the authenticated management channel used by the future Hermes UDM Agent. Device identity credentials are deliberately independent from DDNS credentials: the Agent authenticates with an `hagent_...` key, while `inadyn` continues to authenticate to `/nic/update` with an `hddns_...` key.

## Security properties

- Agent identity keys use the `hagent_<id>.<secret>` format.
- Hermes returns an Agent identity secret only at issuance time and stores only its SHA-256 hash.
- Agent API calls use `Authorization: Bearer <hagent_key>`.
- The credential ID embedded in the key is used for indexed lookup before constant-time hash verification.
- Revoked or expired Agent identity credentials cannot authenticate.
- Successful Agent authentication records `LastUsedAt` and `LastUsedIP`.
- Agent routes derive the Device ID exclusively from the authenticated identity. A Device cannot stage or validate another Device's credential rotation by changing a path parameter.
- DDNS replacement secrets are still generated locally by the Agent. Hermes receives only their Key ID and SHA-256 hash.

## Manual Agent identity bootstrap

26.08-02 provides a temporary administrative bootstrap primitive until one-time enrollment tokens are implemented.

### Issue an identity credential

`POST /api/v1/devices/:id/agent-credentials`

The response includes `agent_key` exactly once. A Device may have only one active Agent identity credential in this milestone.

### List identity credentials

`GET /api/v1/devices/:id/agent-credentials`

Stored hashes and plaintext keys are never returned.

### Revoke an identity credential

`POST /api/v1/devices/:id/agent-credentials/:credential_id/revoke`

After revocation the key immediately stops authenticating. A replacement Agent identity credential can then be issued.

Future enrollment-token support should call the same identity-credential service instead of introducing a second credential format.

## Agent API

All routes below require:

```text
Authorization: Bearer hagent_<id>.<secret>
```

### Identity check

`GET /api/v1/agent/me`

Returns the authenticated Device and Agent credential metadata without secret hashes.

### Discover current credential rotation

`GET /api/v1/agent/credential-rotations/current`

Returns `204 No Content` when no rotation is open. Otherwise it returns the current rotation plus a `next_action` hint.

Typical hints are:

```text
requested   -> generate_and_stage_candidate
staged      -> install_candidate_then_start_validation
validating  -> perform_ddns_update_with_candidate
grace       -> keep_candidate_active_until_grace_completes
```

### Stage a replacement DDNS credential

`POST /api/v1/agent/credential-rotations/:rotation_id/stage`

Request:

```json
{
  "key_id": "0123456789abcdef",
  "secret_hash": "<64-character SHA-256 hex>"
}
```

The Agent generates the complete `hddns_...` secret locally and keeps the plaintext on the UDM. Hermes creates the candidate credential as `pending` using only the Key ID and hash.

### Start validation

`POST /api/v1/agent/credential-rotations/:rotation_id/start-validation`

Hermes changes the staged candidate from `pending` to `active` and the rotation to `validating`. The previous DDNS credential deliberately remains `active` at this point.

The Agent then performs a real `/nic/update` using the candidate key. A successful `good` or `nochg` response confirms the candidate, changes the old credential to `grace`, and starts the grace timer.

## End-to-end flow

```mermaid
sequenceDiagram
  participant Admin
  participant Hermes
  participant Agent
  participant inadyn

  Admin->>Hermes: Request DDNS rotation
  Hermes-->>Agent: current rotation = requested
  Agent->>Agent: Generate hddns_ key locally
  Agent->>Hermes: Stage Key ID + SHA-256 hash
  Hermes->>Hermes: Candidate = pending
  Agent->>Agent: Prepare/install candidate in inadyn
  Agent->>Hermes: Start validation
  Hermes->>Hermes: Candidate = active, rotation = validating
  Note over Hermes: Previous DDNS credential remains active
  Agent->>inadyn: Trigger update with candidate
  inadyn->>Hermes: /nic/update using candidate
  Hermes-->>inadyn: good/nochg
  Hermes->>Hermes: Candidate confirmed; previous = grace
```

## Deferred enrollment work

This block does **not** yet implement the one-time enrollment-token sequence from the architecture specification. The next Agent/enrollment milestone can replace manual Agent credential issuance with:

1. Admin creates a short-lived one-time enrollment token.
2. A newly installed UDM Agent registers using that token.
3. Hermes issues the Device identity credential once over the enrollment response.
4. The Agent persists it securely and uses the Bearer-authenticated API defined in this document.
