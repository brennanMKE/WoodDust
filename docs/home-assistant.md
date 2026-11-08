# Checking for new feedback from Home Assistant

Use the Dust Almanac API to show how many submissions (new species, corrections, sources and reaction reports) are waiting for review, and to get a notification when a new one arrives. Full API reference: [`api.md`](api.md).

## The key

| | |
|---|---|
| Stored in | `~/Developer/Homelab/.env` as `DUST_ALMANAC_API_TOKEN` |
| Site | **Production**, `https://www.dustalmanac.com`. Keys are per environment: this key does not work on pre-prod (`wooddust.sstools.co`), and a pre-prod key does not work here |
| Prefix | `da_nLmXOnsg`, as shown in `/admin/api-keys` |
| Scope | `submissions:read`, which is enough for everything on this page. It can't change anything |

Checked 2026-11-08: `GET /submissions` returns 200 with this key. `GET /stats` returns 403 because the key doesn't have `stats:read`; see [Optional: the stats endpoint](#optional-the-stats-endpoint).

To rotate it, create a new key at https://www.dustalmanac.com/admin/api-keys with the scope `submissions:read`, update `.env` and Home Assistant's `secrets.yaml`, then revoke the old prefix.

## The call

```
GET https://www.dustalmanac.com/api/v1/submissions?status=new&limit=200
Authorization: Bearer <DUST_ALMANAC_API_TOKEN>
```

- `status=new` returns submissions nobody has looked at yet, newest first. Spam is never included.
- When you open one in the admin console and move it to `triaging` (or any other status), it drops out of this list. So **the count is your unread count**.
- `limit` maxes out at 200, so the count tops out at 200.

Response:

```json
{
  "submissions": [
    {
      "id": 7,
      "reference": "DA-0007",
      "kind": "new_species",
      "status": "new",
      "target_slug": null,
      "species_name": "Bubinga",
      "hazard_type": "skin_sensitizer",
      "details": "…",
      "email": "…",
      "created_at": "2026-11-08T21:40:12Z",
      "updated_at": "2026-11-08T21:40:12Z"
    }
  ],
  "next_cursor": null
}
```

The record has more fields than this; Home Assistant needs only `reference`, `kind`, `species_name` or `target_slug`, and `created_at`.

`kind` is one of these values:

| `kind` | Means |
|---|---|
| `new_species` | Suggest a species (`species_name` is set) |
| `correction` | Correction to a card (`target_slug` is set) |
| `new_source` | A source for a card |
| `reaction_report` | Someone reacted to a wood |

**Privacy:** `details` and `email` are free text from the public. Don't copy them into Home Assistant attributes, because the recorder keeps attributes in its database. The config below uses only the reference, kind, species and time.

## Try it from the Mac

```sh
set -a; . ~/Developer/Homelab/.env; set +a
curl -s -H "Authorization: Bearer $DUST_ALMANAC_API_TOKEN" \
  "https://www.dustalmanac.com/api/v1/submissions?status=new&limit=200" \
  | jq '{new: (.submissions | length), newest: (.submissions[0] // {} | {reference, kind, species_name, target_slug, created_at})}'
```

## Home Assistant configuration

`secrets.yaml`. The value includes the word `Bearer`:

```yaml
dust_almanac_api_auth: "Bearer da_nLmXOnsg_…"   # the full DUST_ALMANAC_API_TOKEN value
```

`configuration.yaml`, or a package file:

```yaml
rest:
  - resource: https://www.dustalmanac.com/api/v1/submissions?status=new&limit=200
    headers:
      Authorization: !secret dust_almanac_api_auth
      Accept: application/json
    scan_interval: 900          # 15 minutes; don't go below ~60 s (the API rate-limits per IP)
    timeout: 20
    sensor:
      - name: Dust Almanac new submissions
        unique_id: dust_almanac_new_submissions
        icon: mdi:inbox-arrow-down
        state_class: measurement
        value_template: "{{ value_json.submissions | count }}"

      - name: Dust Almanac latest submission
        unique_id: dust_almanac_latest_submission
        icon: mdi:email-newsletter
        value_template: >-
          {% set s = value_json.submissions | first | default(none) %}
          {{ s.reference if s else 'none' }}
        json_attributes_path: "$.submissions[0]"
        json_attributes:
          - kind
          - species_name
          - target_slug
          - created_at
```

This gives you two entities:
- `sensor.dust_almanac_new_submissions`: the unread count.
- `sensor.dust_almanac_latest_submission`: the newest reference, such as `DA-0007`, with `kind`, `species_name`, `target_slug` and `created_at` as attributes.

If the call fails (bad key, site down), both become `unavailable`.

## Notify when something new arrives

```yaml
automation:
  - alias: Dust Almanac – new submission
    id: dust_almanac_new_submission
    mode: queued
    trigger:
      - platform: state
        entity_id: sensor.dust_almanac_latest_submission
    condition:
      - condition: template
        value_template: >-
          {{ trigger.from_state is not none
             and trigger.from_state.state not in ['unknown', 'unavailable']
             and trigger.to_state.state not in ['unknown', 'unavailable', 'none'] }}
    action:
      - variables:
          a: "{{ trigger.to_state.attributes }}"
          labels:
            new_species: "Species suggestion"
            correction: "Correction"
            new_source: "New source"
            reaction_report: "Reaction report"
      - service: notify.notify            # or notify.mobile_app_<your_phone>
        data:
          title: "Dust Almanac: {{ labels.get(a.kind, a.kind) }}"
          message: >-
            {{ trigger.to_state.state }}
            {%- if a.species_name %} · {{ a.species_name }}{% endif %}
            {%- if a.target_slug %} · {{ a.target_slug }}{% endif %}
            ({{ states('sensor.dust_almanac_new_submissions') }} waiting)
          data:
            url: "https://www.dustalmanac.com/admin/submissions"
            clickAction: "https://www.dustalmanac.com/admin/submissions"
```

How it behaves:
- **It triggers when the newest reference changes**, so a new submission notifies once, even if older ones are still unread.
- **No notification at startup, or when the site comes back** after being unreachable, because of the `from_state` checks.
- **Several submissions within one 15-minute poll** produce one notification. The count shows how many are waiting.
- **Tapping the notification** opens the admin list, where you sign in with your passkey.

For a dashboard card, show `sensor.dust_almanac_new_submissions`, for example with a conditional card that only appears when it's above 0.

## Optional: the stats endpoint

`GET /api/v1/stats` returns everything in one small response:
- open counts by status and by kind;
- `newest_at`, `new_last_24h`, `spam_last_24h`.

It never includes submission text. It needs `stats:read`, which this key doesn't have. To use it, create a second key with only `stats:read`; the config is in [`api.md` § Home Assistant](api.md#home-assistant). Keeping `submissions:read` for the alert above is still useful, because only it can tell you the reference and species.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `401` / sensors `unavailable` | Key missing, mistyped (the secret must start with `Bearer `), revoked, expired, or used against pre-prod |
| `403` | The key lacks the scope; `/stats` needs `stats:read` |
| `429` | Polling too often; raise `scan_interval` |
| Count never drops | The submissions are still `new`. Move them to `triaging` (or another status) in the admin console |
