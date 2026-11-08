# Deployment

Interim host: `wooddust.sstools.co` on the existing EC2 instance (`ssh photon`), behind Apache with the `sstools.co-wildcard` certificate. Nothing here has been deployed yet; this is the intended setup.

## Layout on the host

| Path | Contents |
|---|---|
| `/opt/wooddust/releases/<git-sha>/` | A checkout of the repo at that commit plus `bin/wooddust` built from it (SPA embedded) |
| `/opt/wooddust/current` | Symlink to the live release |
| `/etc/wooddust/config.env` | Environment (see `deploy/config.env.example`) |
| `/etc/systemd/system/wooddust.service` | From `deploy/systemd/wooddust.service` |
| `/etc/httpd/conf.d/wooddust.sstools.co-le-ssl.conf` | From `deploy/apache/` |

The service reads `data/`, `content/` and `images/` from its working directory (`/opt/wooddust/current`), so a release is self-contained and rollback is a symlink flip.

Port: **8084** (8080–8083 are taken by other services on the host). The app always binds `127.0.0.1`.

## One-time setup

```sh
sudo useradd --system --home /opt/wooddust --shell /sbin/nologin wooddust
sudo mkdir -p /opt/wooddust/releases /etc/wooddust
sudo -u postgres createuser wooddust
sudo -u postgres createdb -O wooddust wooddust
# pg_trgm and unaccent are trusted extensions (PG 13+): the owner can create them.
sudo install -m 0640 -o root -g wooddust deploy/config.env.example /etc/wooddust/config.env   # then edit
sudo install -m 0644 deploy/systemd/wooddust.service /etc/systemd/system/
sudo install -m 0644 deploy/apache/wooddust.sstools.co-le-ssl.conf /etc/httpd/conf.d/
sudo systemctl daemon-reload && sudo systemctl enable wooddust
sudo httpd -t && sudo systemctl reload httpd
```

DNS: `wooddust.sstools.co` is covered by the wildcard record and certificate; no new certificate is needed. The existing certbot deploy hook reloads Apache on renewal.

## Sign-in and submissions (one-time, owner to approve)

Written for the owner to review and run. Nothing here has been done on the host.

Sign-in uses the Portiko provider embedded in the binary (PLAN §5, `docs/portiko-integration.md`). Portiko's tables live in a **separate database** on the same Postgres server. `wooddust serve` applies Portiko's migrations to it on every start, as it does for the site database.

1. Create the identity database:

   ```sh
   sudo -u postgres createdb -O wooddust wooddust_identity
   ```

2. Generate the secrets, one `openssl rand -hex 32` each, and put them in `/etc/wooddust/config.env` (see `deploy/config.env.example`):

   | Variable | Purpose |
   |---|---|
   | `IDENTITY_CRYPTO_KEY` | Portiko: seals the signing keys at rest and encrypts authorization codes. Must stay stable; losing it means re-enrolling |
   | `IDENTITY_CLIENT_SECRET` | The site's client secret with its own provider |
   | `COOKIE_KEY` | Protects the sign-in redirect cookies (state, PKCE, nonce) |
   | `FORM_TOKEN_KEY` | Signs submit-form tokens; also derives the daily IP-hash salt unless `IP_HASH_KEY` is set |
   | `IDENTITY_BOOTSTRAP_CODE` | Opens passkey enrollment at `/auth/enroll`. Use another long random value |

   Also set `IDENTITY_DATABASE_URL`, and the `IDENTITY_SMTP_*`/`IDENTITY_EMAIL_FROM` settings for one-time codes (until SMTP is configured, codes go to the journal, which is fine for the one owner enrollment but must not stay that way). Do **not** set `DEV_INSECURE` in production. `TRUSTED_PROXIES=127.0.0.1,::1` is shared with Portiko's rate limits.

3. Register the site as the provider's single first-party client. This is a one-time command, run as the service user with the service's environment file (`systemd-run` reads it exactly as the unit does, including values with spaces):

   ```sh
   wd() { sudo systemd-run --pipe --wait --quiet -p User=wooddust \
     -p EnvironmentFile=/etc/wooddust/config.env -p WorkingDirectory=/opt/wooddust/current \
     /opt/wooddust/current/bin/wooddust "$@"; }
   wd identity register-site
   ```

   The `wd` helper is used for the other operator commands below too.

   `register-site` is shorthand for Portiko's own command with this site's values:

   ```sh
   wooddust identity register-client -first-party wooddust https://wooddust.sstools.co \
     https://wooddust.sstools.co/auth/callback "$IDENTITY_CLIENT_SECRET"
   ```

   Optional, so a sign-out at the provider also ends the site session:
   `wooddust identity set-logout-uri wooddust https://wooddust.sstools.co/auth/backchannel-logout`.

4. Restart the service. The log should say `sign-in enabled`.

5. **Enroll the owner** (once, from your own browser):
   1. Open `https://wooddust.sstools.co/auth/enroll` and enter `IDENTITY_BOOTSTRAP_CODE`. That sets a 30-minute cookie that opens Portiko's registration endpoints for this browser only. Without it they answer 403, so nobody else can create an account.
   2. You are sent to sign in. Choose "sign in with your email", enter the code from the email (or from `journalctl -u wooddust` before SMTP is set up), and create a passkey.
   3. You land on `/admin`. **The first subject ever to sign in becomes the admin** (the local `users.is_admin`). Anyone who signs in later gets 403 on every admin route.
   4. Check with `wd users list`. Afterwards you may remove `IDENTITY_BOOTSTRAP_CODE`, which closes enrollment completely. Keep it in your password manager to re-open enrollment for a new device or recovery.

   If the wrong subject became admin: `wd users unadmin <subject>` and `wd users admin <subject>`.

6. API keys (e.g. for Home Assistant, see `docs/api.md`):

   ```sh
   wd apikey create -name home-assistant -scopes stats:read
   wd apikey list
   wd apikey revoke <prefix>
   ```

The RP ID is the host of `BASE_URL`. A passkey is bound to it forever, so moving to a purchased domain means enrolling again (PLAN §10), and re-running `register-site` for the new `BASE_URL`.

Backups: once submissions exist, dump both `wooddust` and `wooddust_identity` nightly.

## Each release

Build locally (or in CI) for linux/arm64 or amd64 as the instance requires:

```sh
make web                                       # npm ci && npm run build -> web/dist
GOOS=linux GOARCH=arm64 go build -trimpath -o bin/wooddust ./cmd/wooddust
```

On the host:

```sh
SHA=<git sha>
sudo -u wooddust git clone --depth 1 <repo> /opt/wooddust/releases/$SHA   # or git archive | tar
scp bin/wooddust photon:/tmp/wooddust && sudo install -o wooddust -m 0755 /tmp/wooddust /opt/wooddust/releases/$SHA/bin/wooddust
sudo ln -sfn /opt/wooddust/releases/$SHA /opt/wooddust/current
sudo systemctl restart wooddust      # ExecStartPre runs `wooddust seed` (migrates, then reseeds)
curl -fsS https://wooddust.sstools.co/api/v1/health
```

`serve` and `seed` both apply pending migrations first. Migrations are append-only once applied here. `seed` refuses to run if the data fails its integrity check, and the service then stays on the previous process's failure path (it will not start), so check `journalctl -u wooddust` after a release. Run `wooddust seed -check` before deploying (CI does).

Rollback: point `current` at the previous release and restart. Published data is reseeded from that release's `data/`.

## Images

Apache serves `/images/` directly from `current/images/` with a 30-day cache; the Go server's `/images/` handler is only for development. Files whose names start with `_` and `images/prompts/` are not served. When a species or page image is missing, the API points at `/images/placeholder.svg` and share cards fall back to `/images/og-default.jpg` (once it exists), then the placeholder. Renaming a file is how you bust the cache.

## Later

- Moving to a purchased domain: add a vhost for it, set `BASE_URL`, and 301 the old host (PLAN §10).
- Backups: dump the `wooddust` and `wooddust_identity` databases nightly once they hold submissions; published tables are rebuildable from git.
