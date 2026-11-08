# Dust Almanac REST API

Base: `https://<domain>/api/v1`. JSON only (`application/json; charset=utf-8`). The binding spec is `docs/contract.md` §3, §6 and §7; this page is the practical guide.

Errors always look like:

```json
{ "error": { "code": "not_found", "message": "...", "fields": { "details": "..." } } }
```

`fields` appears only on `422 invalid`. Every `/api/` request is rate-limited per client IP (`API_RATE_PER_MIN`, default 120/min, burst 60); over the limit you get `429` with `Retry-After`.

## Public (no key)

| Method and path | Returns |
|---|---|
| `GET /species[?featured=true&wood_type=&hazard_type=&evidence_level=]` | `{ "species": [summary…] }` |
| `GET /species/{slug}` | Full species record with its sources and the regulatory summary |
| `GET /search?q=&limit=8` | `{ "query", "results": [...] }` |
| `GET /sources`, `GET /sources/{key}` | Source registry |
| `GET /pages/{path}` | Rendered education page |
| `GET /health` (also `/health`) | `{ "status": "ok", "species": n, "sources": n }` |

### Submissions (anonymous)

| Method and path | Notes |
|---|---|
| `GET /submissions/form-token` | `{ "form_token": "..." }`. Signed with the issue time. It must be at least 4 s old (`SUBMIT_MIN_FILL_SECONDS`) and at most 24 h old (`SUBMIT_FORM_MAX_AGE_HOURS`) when posted |
| `GET /submissions/check-url?url=` | `{ "warnings": [...], "leads_only": bool }`. Warns for leads-only domains (see `CONTRIBUTING.md`) |
| `POST /submissions` | Body per contract §6. `201 { id, reference, status_url, warnings }`, `422` with `fields`, `429` with `Retry-After` |
| `GET /submissions/status/{token}` | `{ reference, kind, status, created_at, updated_at, notes[] }`. 404 for an unknown token |

Abuse controls (PLAN §6), all server-side:

- Ceilings: 5 per IP per hour, 20 per IP per day, and 200 per day site-wide. Configure them with `SUBMIT_PER_IP_HOUR`, `SUBMIT_PER_IP_DAY` and `SUBMIT_GLOBAL_DAY`. This limiter is separate from the GET limit above.
- A filled `website` honeypot, or a form token younger than the minimum fill time, still gets a normal `201`. The submission is stored as `spam`, and its status page shows `new`.
- A token that is expired or forged gets `422` on `form_token`.
- Size limits: details are 20–4000 characters with at most 5 links. At most 10 citations. No uploads.
- Privacy: the IP is stored only as an HMAC under a salt that rotates daily (`FORM_TOKEN_KEY`/`IP_HASH_KEY`). The status token is 32 random bytes, and only its SHA-256 is stored. The email is set to null when the status becomes `rejected`, `merged` or `spam`.

## Authenticated

Send an API key as `Authorization: Bearer da_<prefix>_<secret>`, or use the owner's signed-in session (the admin console). Session requests with unsafe methods must carry a same-origin `Origin` header. API-key requests need none.

| Method and path | Scope |
|---|---|
| `GET /submissions?status=&kind=&since=&limit=&cursor=` → `{ "submissions": [...], "next_cursor": "…" \| null }` | `submissions:read` |
| `GET /submissions/{id}` → the record at the top level plus `events[]` (`at`, `type`, `actor`, `from_status`, `to_status`, `note`, `visible_to_submitter`) | `submissions:read` |
| `PATCH /submissions/{id}` with `{ "status"?, "note"?: { "text", "visible_to_submitter" } }` | `submissions:write` |
| `GET /submissions/{id}/export` | `submissions:read` |
| `GET /stats` | `stats:read` |
| `GET /admin/api-keys` → `{ "keys": [...] }`, `POST /admin/api-keys`, `DELETE /admin/api-keys/{prefix}` | session only |
| `GET /me` → `{ "user": { "display_name", "is_admin" } }` | session |

Notes:

- `{id}` accepts `42` or `DA-0042`.
- The list is newest first. With no `status`, it excludes `spam`; pass `status=spam` to review spam. `since` is RFC 3339 or `YYYY-MM-DD`. `limit` defaults to 50 (max 200). Pass `next_cursor` back as `cursor` for the next page.
- Every status change or note writes a `submission_events` row and an `audit_log` row naming the actor (`user:<subject>` or `key:<prefix>`). Creating and revoking keys is audited too.
- **Export** works only for `accepted`, `exported` or `merged` submissions. Reaction reports never export (409). The response is a **draft for human editing** (`"draft": true` plus a `notice`). It contains:
  - `action`: `create` or `patch`;
  - `file`: `data/species/<slug>.json`;
  - `species`: the proposed file, or the current record with changes applied;
  - `changes`: what changed, in words;
  - `proposed_sources`: under `TODO-DA-0042-n` keys;
  - the submitter's text.

  Every summary must be rewritten in our own words and every source verified before a pull request.

### `GET /stats`

```json
{
  "open": 3,
  "by_status": { "new": 2, "triaging": 0, "needs_info": 1, "accepted": 0, "rejected": 4, "exported": 0, "merged": 7, "spam": 12 },
  "by_kind": { "new_species": 1, "correction": 1, "new_source": 0, "reaction_report": 1 },
  "newest_at": "2026-11-08T09:12:44Z",
  "new_last_24h": 1,
  "spam_last_24h": 3
}
```

- `open` counts everything not `rejected`, `merged` or `spam`.
- `by_kind` counts open submissions only.
- `by_status` covers all statuses, including spam.
- `newest_at` and `new_last_24h` ignore spam.

## API keys

Format: `da_` + an 8-character base62 prefix + `_` + 32 random bytes in base62. The full key is shown **once**. Only the SHA-256 is stored, and it is compared in constant time. Each use updates `last_used_at`.

Scopes: `submissions:read`, `submissions:write`, `stats:read`.

From the admin console (`/admin/api-keys`) or the CLI:

```sh
wooddust apikey create -name home-assistant -scopes stats:read [-expires 2027-12-31]
wooddust apikey list
wooddust apikey revoke <prefix>
```

## Home Assistant

Step-by-step setup with the production key, a new-submission notification and troubleshooting: [`home-assistant.md`](home-assistant.md). The minimal stats-only version:

A `stats:read` key is enough. In `secrets.yaml`:

```yaml
wooddust_api_key: "Bearer da_XXXXXXXX_..."
```

In `configuration.yaml`:

```yaml
rest:
  - resource: https://<domain>/api/v1/stats
    headers:
      Authorization: !secret wooddust_api_key
    scan_interval: 900
    sensor:
      - name: Dust Almanac open submissions
        unique_id: dust_almanac_open_submissions
        value_template: "{{ value_json.open }}"
        json_attributes:
          - by_status
          - by_kind
          - newest_at
          - new_last_24h
          - spam_last_24h
      - name: Dust Almanac new submissions (24h)
        unique_id: dust_almanac_new_24h
        value_template: "{{ value_json.new_last_24h }}"
```

An automation can notify on a state change of `sensor.dust_almanac_new_submissions_24h`. Avoid `scan_interval` below about 60 s, because the per-IP API limit applies to these calls too.

## Quick check with curl

```sh
TOKEN=$(curl -s https://<domain>/api/v1/submissions/form-token | jq -r .form_token); sleep 5
curl -s -X POST https://<domain>/api/v1/submissions -H 'Content-Type: application/json' -d "{
  \"kind\":\"new_species\",\"species_name\":\"Bubinga\",\"details\":\"Reported skin sensitizer in a 2019 case series.\",
  \"citations\":[{\"doi\":\"10.1111/cod.13000\"}],\"form_token\":\"$TOKEN\",\"website\":\"\"}"
curl -s -H "Authorization: Bearer $KEY" https://<domain>/api/v1/stats
```
