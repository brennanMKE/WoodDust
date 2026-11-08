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
- Portiko sign-in will need its own database (`wooddust_identity`) and secrets in `config.env` (PLAN §5).
- Backups: dump the `wooddust` database nightly once it holds submissions; published tables are rebuildable from git.
