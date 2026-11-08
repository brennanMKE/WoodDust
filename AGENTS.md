# AGENTS.md — Dust Almanac

Binding guidance for any agent (Claude, Codex, others) working in this repository. Read it before changing anything.

**Authoritative sources:**

| For | Read |
|---|---|
| Product scope, data model, phases, open questions | [`PLAN.md`](PLAN.md) |
| Data policy: what counts as evidence, sourcing tiers, tone | [`wood-dust-site-handoff.md`](wood-dust-site-handoff.md) §1–5 and [`data/RESEARCH_BRIEF.md`](data/RESEARCH_BRIEF.md) |
| Backend ↔ SPA contract (routes, API shapes, SSR markers) | [`docs/contract.md`](docs/contract.md). Change it only in the same commit as both sides |
| Hosting, environments, scripts | [`docs/deployment.md`](docs/deployment.md); photon itself: `~/Developer/AWS` (`AddSite.md`, `AccessLogs.md`, port registry) |
| Embedded Portiko sign-in and its workarounds | [`docs/portiko-integration.md`](docs/portiko-integration.md); upstream asks: [`docs/portiko-requests.md`](docs/portiko-requests.md) |
| REST API and keys | [`docs/api.md`](docs/api.md) |
| Illustration and photo prompts | [`images/prompts/site-art.md`](images/prompts/site-art.md), [`images/prompts/codex-art.md`](images/prompts/codex-art.md), species `image_prompt` in `data/species/*.json` |

## 1. What this is

A wood-dust hazard reference: a card per species, education pages and anonymous submissions, plus an owner-only admin console and API.
- **Backend:** a Go server (`cmd/wooddust`) with Postgres. It serves a Svelte 5 SPA (`web/`) embedded in the binary, server-rendering each page's head and body for SEO.
- **Sign-in:** passkeys through Portiko, embedded in the same binary.
- **Data:** species data lives as JSON in git (`data/`) and is seeded into Postgres on every start.

| Path | Contains |
|---|---|
| `cmd/wooddust/` | Binary: `serve`, `seed [-check]`, `migrate`, `identity …`, `apikey …`, `users …` |
| `internal/` | `config`, `db`, `site` (SSR, API), `identity` (Portiko), `contrib` (submissions, keys, sessions) |
| `migrations/` | golang-migrate SQL, **append-only** (applied in pre-prod and prod) |
| `data/` | `species/*.json`, `sources.json`, `regulatory.json`, schemas, `verify/` (unverified items) |
| `content/pages/*.md` | Education pages (front matter + Markdown, `[^KEY]` citations) |
| `web/` | Svelte 5 + Vite SPA; `npm run build` → `web/dist` (embedded) |
| `images/` | `species/<slug>.jpg`, `pages/`, `art/`, `mascots/`, `og-default.jpg`, `prompts/` (never served) |
| `deploy/` | `envs/{preprod,prod}.env`, Apache vhosts per env, `systemd/site.service.in`, `config.env.example` |
| `scripts/` | See §2 |

## 2. Scripts: use these, don't improvise

| Task | Command |
|---|---|
| Run locally with Postgres (`wooddust_dev`) | `make dev` → http://localhost:8080 |
| Run locally without Postgres | `make dev-memory` |
| SPA hot reload | `make dev`, plus `cd web && npm run dev` |
| SPA without a backend (fixtures) | `cd web && npm run dev:fixtures` |
| **Verify before committing** | `make check` (`scripts/check.sh`: vet, tests, schema, integrity; must print `VERIFICATION PASSED`) |
| Data integrity only | `make seed-check` |
| Host setup for an environment (idempotent) | `make provision-preprod` / `make provision-prod` |
| Deploy the current commit | `make deploy-preprod` / `make deploy-prod` |
| Deploy a specific commit | `scripts/deploy.sh <env> --ref <sha>` |
| What's live, logs | `make status-<env>`, `make logs-<env>` |
| Roll back | `make rollback-<env>`, or `scripts/rollback.sh <env> <sha>` |

**Environments, both on photon:**
- **preprod:** https://wooddust.sstools.co. Sends `noindex`.
- **prod:** https://www.dustalmanac.com. The apex redirects to www.

They differ only in `deploy/envs/*.env`.
- **Deploys ship commits only:** `deploy.sh` builds from `git archive`, never from the working tree, so commit first.
- **Prod only takes pushed commits:** it deploys only commits on `origin/main` and asks for a typed confirmation unless `--yes`.
- **Failed deploys roll back automatically** to the previous release.

## 3. Rules

### Servers (photon, AWS)
- **Do nothing on photon or in AWS (ssh, scp, deploy, provision, DNS, certificates, database changes) without Brennan's explicit approval of that specific step.** Approval for one step doesn't cover the next. Read-only checks need approval too unless Brennan says otherwise. Write the steps down and ask.
- **Use the scripts** for server work. If a script can't do what's needed, change the script, not the host by hand.
- **Never print, copy off the host, or commit** `config.env` or any secret. Secrets are generated on the host by `provision.sh`.
- **Follow photon's conventions:**
  - ports only through `/etc/photon/ports.conf` + `photon-ports`;
  - vhosts as `001-<host>` pairs, logs under `logs/sites/<host>/` with `vhostlog`;
  - **`apachectl configtest` before every reload**, because a broken reload takes every site on the box down.
- **Never run `certbot --apache`.** See `AddSite.md`.

### Code
- **Stack:** stdlib `net/http` with Go 1.22+ routing, `pgx/v5`, `slog`. No web framework.
- **Migrations are append-only.** Add a new one; never edit an applied one.
- **Keep the CSP strict:** no inline `<script>`/`<style>` or `style=""` in the SPA or SSR output. The CSP is `'self'`; only Portiko's own paths are looser.
- **Contract changes:** update `docs/contract.md` in the same commit as the Go and Svelte sides.
- **Before handing off,** run `make check` and `cd web && npx svelte-check`.

### Data and content
- **Stay within the evidence:** follow the handoff's data policy. Never fill gaps from memory; unverifiable items go in `data/verify/*.md`.
- **Leads, never citations:** The Wood Database, Haz-Map, forums and magazine charts.
- **Write summaries in your own words:** calm, plain language, never pasted abstracts.
- **Evidence levels:** 1 confirmed in clinical testing · 2 listed by an agency · 3 reported · 0 applies to all wood dust. Badges always show the text label.
- **Every species keeps the level-0 IARC carcinogen entry.**
- **Validate data changes** with `make seed-check`; CI runs the same check.

### Images
- **Species photos:** `images/species/<slug>.jpg`, 1200×630 JPEG, under 300 KB, from the species `image_prompt`.
- **Illustrations:**
  - follow `images/prompts/codex-art.md`: exact paths and sizes, real alpha transparency where specified;
  - no text in images;
  - **Paul is always clean-shaven with a trimmed mustache and wears safety gear correctly**;
  - nothing may resemble the Bemidji or Brainerd statues, Disney's version, or any specific vintage artwork.
- **Leave generated images unpublished:** don't commit them until Brennan approves them.

### Git
- **Commit and push only when asked.** Commit messages are imperative and describe the change.
- **Never commit:** `.env`, `bin/`, `web/dist/*` (except `.gitkeep`), `node_modules/`.
