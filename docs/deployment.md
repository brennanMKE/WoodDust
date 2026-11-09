# Deployment

Two environments run on photon (`ssh photon`, AWS docs in `~/Developer/AWS`):

| | Pre-prod | Prod |
|---|---|---|
| URL | https://wooddust.sstools.co (sends `X-Robots-Tag: noindex`) | https://www.dustalmanac.com (apex 301s to www) |
| Env file | `deploy/envs/preprod.env` | `deploy/envs/prod.env` |
| Unit / user / Postgres role | `wooddust` | `dustalmanac` |
| Databases | `wooddust`, `wooddust_identity` | `dustalmanac`, `dustalmanac_identity` |
| Root, config | `/opt/wooddust`, `/etc/wooddust/config.env` | `/opt/dustalmanac`, `/etc/dustalmanac/config.env` |
| Port (photon registry) | 8112 | 8113 |
| Certificate | `sstools.co-wildcard` (existing) | `dustalmanac.com` (apex + www, HTTP-01 webroot `/var/www/letsencrypt`) |

Each environment has its own secrets, sign-in provider and passkeys: enroll separately on each.

## Scripts

| Task | Command |
|---|---|
| Run locally (Postgres) | `make dev` (`scripts/dev.sh`); without Postgres: `make dev-memory` |
| Verify before shipping | `make check` |
| One-time host setup, re-run after changing `deploy/` | `make provision-preprod`, `make provision-prod` |
| Deploy HEAD | `make deploy-preprod`, `make deploy-prod` (prod asks you to type `prod`; only commits on `origin/main`) |
| Deploy a specific commit | `scripts/deploy.sh prod --ref <sha>` |
| What is live | `make status-prod`, `make logs-prod` |
| Roll back | `make rollback-prod` (previous release) or `scripts/rollback.sh prod <sha>` |
| Store an operator setting (the ShortLinks key) | `scripts/set-setting.sh <env> <key>`, or `make shortlinks-key-preprod` / `make shortlinks-key-prod` |
| Build the poster print files (local) | `make posters` (`scripts/posters.sh`, needs `rsvg-convert`) |

`deploy.sh` builds from `git archive` of the commit (never the working tree), runs the data integrity check, cross-compiles for arm64, uploads, switches `current`, restarts and waits for health, rolling back automatically if the new release does not come up. The first deploy also registers the site's sign-in client. It keeps the last 5 releases.

`provision.sh` is idempotent: Route 53 records (prod, `admin` profile), user, databases, `config.env` (secrets generated on the host and never printed), port registry, systemd unit (rendered from `deploy/systemd/site.service.in`), log directories, vhosts (config-tested, reverted on failure) and the certificate. On a host that already has `config.env`, it leaves the file alone except for generated keys added since (today `SETTINGS_KEY`), which it appends without printing; restart the unit afterwards.

Why HTTP-01 for prod: photon's instance role may only write the six zones in the AWS repo's `terraform/variables.tf` (`certbot_zones`). Adding `dustalmanac.com` there would allow dns-route53 and a wildcard, but needs that Terraform state.

The sections below describe the original manual setup of pre-prod and remain accurate for what the scripts do.

## Manual reference

Interim host: `wooddust.sstools.co` on the existing EC2 instance (`ssh photon`), behind Apache with the `sstools.co-wildcard` certificate. First deployed 2026-11-08 as a pre-production site.

## Layout on the host

| Path | Contents |
|---|---|
| `/opt/wooddust/releases/<git-sha>/` | A checkout of the repo at that commit plus `bin/wooddust` built from it (SPA embedded) |
| `/opt/wooddust/current` | Symlink to the live release |
| `/etc/wooddust/config.env` | Environment (see `deploy/config.env.example`) |
| `/etc/systemd/system/wooddust.service` | From `deploy/systemd/site.service.in (rendered)` |
| `/etc/httpd/conf.d/001-wooddust.sstools.co{,-le-ssl}.conf` | From `deploy/apache/` (HTTPS vhost and the HTTP redirect) |
| `/etc/photon/ports.conf` | photon's port registry; the `wooddust` row sets the port |

The service reads `data/`, `content/` and `images/` from its working directory (`/opt/wooddust/current`), so a release is self-contained and rollback is a symlink flip.

Port: **8112**, registered in photon's port registry (`/etc/photon/ports.conf`, `sstools.co tools` block). `sudo photon-ports render` writes `PORT` into a systemd drop-in and `PORT_WOODDUST` into Apache's `000-ports.conf`, so `PORT` is never set in `config.env` or the unit. The app always binds `127.0.0.1`.

## One-time setup

```sh
sudo useradd --system --home /opt/wooddust --shell /sbin/nologin wooddust
sudo mkdir -p /opt/wooddust/releases /etc/wooddust
sudo -u postgres createuser wooddust
sudo -u postgres createdb -O wooddust wooddust
# pg_trgm and unaccent are trusted extensions (PG 13+): the owner can create them,
# but they ship in the contrib package, which photon did not have:
sudo dnf install -y postgresql15-contrib   # match the server's major version
sudo install -m 0640 -o root -g wooddust deploy/config.env.example /etc/wooddust/config.env   # then edit
sudo install -m 0644 deploy/systemd/site.service.in (rendered) /etc/systemd/system/
sudo install -m 0644 deploy/apache/preprod/001-wooddust.sstools.co-le-ssl.conf deploy/apache/preprod/001-wooddust.sstools.co.conf /etc/httpd/conf.d/
sudo install -d -m 0700 /var/log/httpd/sites/wooddust.sstools.co
# Add to /etc/photon/ports.conf under "8110-8119: sstools.co tools":
#   8112    wooddust               loopback  wooddust.sstools.co
sudo photon-ports render          # drop-in + Apache define, daemon-reload, configtest
sudo systemctl enable wooddust
sudo httpd -t && sudo systemctl reload httpd
sudo photon-ports check
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
   | `SETTINGS_KEY` | Encrypts secret operator settings in the database (the ShortLinks key). Exactly 32 bytes. Losing it means setting those again |

   Also set `IDENTITY_DATABASE_URL`, and the `IDENTITY_SMTP_*`/`IDENTITY_EMAIL_FROM` settings for one-time codes; see Email (Amazon SES) below. Without SMTP, codes go to the journal. Do **not** set `DEV_INSECURE` in production. `TRUSTED_PROXIES=127.0.0.1,::1` is shared with Portiko's rate limits.

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

## Poster links: the ShortLinks key

Plaque posters get their own QR code link from the go.dustalmanac.com ShortLinks instance (`docs/posters.md`). Without a key, they use the stock link, so this is optional.

1. Make sure `SETTINGS_KEY` is in `config.env` (`make provision-<env>` adds it) and the unit was restarted.
2. On go.dustalmanac.com, create an API token under Account › API tokens: **not** read-only, 12-month expiry, one per environment. Create the `posters` campaign there too.
3. From this Mac: `scripts/set-setting.sh preprod shortlinks_api_key`. It asks for the token with hidden input (or reads it from a pipe; prod then needs `--yes`) and sends it over ssh stdin to `wooddust settings set shortlinks_api_key --stdin`, run with `systemd-run` as the service user and its `config.env`, like `register-site`. The token never appears in arguments, the environment or any output; the command prints only `slk_AbCdEfGh…`.
4. If go.dustalmanac.com doesn't listen on the default loopback port, set it the same way: `scripts/set-setting.sh <env> shortlinks_api_url` (e.g. `http://127.0.0.1:<port>` from `/etc/photon/ports.conf`).
5. Open `/admin/settings` and press **Test connection**. The dashboard shows a banner until it says OK.

To rotate: create the new token, run step 3 again, test, then revoke the old token in ShortLinks.

## Email (Amazon SES)

Sign-in codes go through Amazon SES SMTP (set up 2026-11-08, following `~/Developer/AWS/SES.md`):

- **Identity:** `dustalmanac.com` (DKIM, custom MAIL FROM `mail.dustalmanac.com`, SPF, DMARC `p=quarantine`; records in its Route 53 zone). The account is out of the SES sandbox.
- **Credentials:** IAM user `ses-smtp-dustalmanac`, whose inline policy allows only `ses:SendRawEmail`/`SendEmail` on that identity with `ses:FromAddress = noreply@dustalmanac.com`. Its SMTP username and password are in both `config.env` files (`IDENTITY_SMTP_*`) and nowhere else.
- **From address:** `IDENTITY_EMAIL_FROM=noreply@dustalmanac.com` for pre-prod and prod. It must be a bare address, because Portiko uses it for the header and the SMTP envelope.
- **Network:** the unit no longer sets `IPAddressDeny`, since SES endpoints have no fixed IPs.
- **Rotating the key:**
  1. `aws iam create-access-key --user-name ses-smtp-dustalmanac`.
  2. Derive the SMTP password with `~/Developer/AWS/ses/derive-smtp-password.py <secret> us-east-1`.
  3. Update both config files and restart both units.
  4. Delete the old key.

## Each release

Build locally (or in CI) for linux/arm64 or amd64 as the instance requires:

```sh
make web                                       # npm ci && npm run build -> web/dist
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o bin/wooddust-linux-arm64 ./cmd/wooddust   # photon is aarch64
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
