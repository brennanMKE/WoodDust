# Portiko integration

Dust Almanac embeds the Portiko identity provider in its own binary and uses Portiko's `rp` and `rp/session` packages for the site's session (PLAN §5). Only the owner signs in, and only for `/admin`. This page records how it is wired, what did not fit Portiko's assumptions at the pinned commit, and what we did about each mismatch without forking.

Pinned: `github.com/Kolonnade/Portiko v0.0.0-20261106224428-4d9d3082a998` (`main` at 2026-11-06). Code: `internal/identity`. Tests: `internal/identity/identity_test.go` drives the real provider with `descope/virtualwebauthn`. It covers the enrollment gate, passkey registration, sign-in, admin and non-admin users, the CSRF check, API key creation, and full sign-out.

## Wiring

| Piece | How |
|---|---|
| Provider config | Built in code (`identity.ProviderConfig`) from the site config, never from Portiko's env loader. Issuer = `BASE_URL`; RP ID = its host; RP origins = `BASE_URL`; `ServiceName` = `SITE_NAME`; `TRUSTED_PROXIES` shared; `IDENTITY_*` for the database, crypto key, bootstrap code and SMTP; `DEV_INSECURE` for http://localhost |
| Database | Separate: `IDENTITY_DATABASE_URL` (`wooddust_identity`, `wooddust_identity_dev`, `wooddust_identity_test`) |
| Routes | `p.Handler()` is mounted only on `/oauth/`, `/accounts/`, `GET /.well-known/{openid-configuration,jwks.json,portiko-configuration,eureka-accounts-configuration}`, `GET /login`, `POST /login/select`, `GET /signin`, `GET /u/`. The site keeps `/`, `/health` and `/profile` |
| Site session | `rp.Client` with the Postgres `session.Store` in `internal/contrib/sessions.go` (`rp_sessions`; only the SHA-256 of the session id is stored). `/auth/login`, `/auth/callback`, `/auth/logout`, `/auth/backchannel-logout` |
| Authorization | Local `users` row keyed by subject. `OnSignIn` upserts it; the first subject ever becomes `is_admin` (advisory-locked). Others get 403 |
| Lifecycle | `go id.Sweep(ctx)` and `defer id.Close()` in `serve`; expired `rp_sessions` swept hourly |
| Operator commands | `wooddust identity <cmd>` maps `IDENTITY_*` onto Portiko's variable names, then calls `server.Main`. It also adds `migrate` and `register-site` (this site as the first-party client). `serve` is refused, because the provider is already embedded |
| Disabled | Without `IDENTITY_DATABASE_URL` (or without `DATABASE_URL`), `/auth/*` and `/admin` answer 503 with the reason. Session-only API endpoints answer 503, and API keys still work |

## Mismatches and workarounds

### 1. The router assumes it owns `/`

Portiko's router registers `GET /{$}` (the account home, which redirects to `/u/{n}/`), `GET /health` and `GET /profile`. We do not mount those three. What changes as a result:

- **No account page at `/`.** Portiko's pages, scripts and redirects send people to `/` in these cases:
  - `signin.html` "Back to your accounts" (only when adding a second account);
  - the redirect when `/signin` is opened while already signed in with no `return_to`;
  - a form-post logout from the account page when other accounts remain;
  - the `auth-watch` script's `signedOut: location.replace('/')` on the home and profile pages;
  - the fallback `window.location = … || '/'` after a ceremony with no return path.

  In each case the person lands on the Dust Almanac home page instead of Portiko's account home. The account home is still at `/u/0/` and the profile at `/u/0/profile`. **Nothing breaks.** The owner only ever reaches the provider through `/auth/login`, which always carries an authorization request and returns to `/admin`. Fix: none needed. Least invasive is to leave it.
- **`/profile`** (Portiko's "sites link here without knowing numbers") is the site's, and is 404 today. Nothing in Dust Almanac links to it.
- **`/health`** is the site's. It does not ping the identity database. Portiko's ping is not reachable, which is acceptable because `serve` fails at startup if the identity database is unreachable.
- `DefaultLogoutRedirectURI` is `<issuer>/signin`, which is mounted, so that works.

A path-prefix option in Portiko (e.g. serving everything under `/id/`) would remove all of this. Worth an upstream issue, per PLAN §5. It is not needed to ship.

### 2. Inline scripts versus the site CSP

The site CSP is `script-src 'self'` with no inline scripts. Portiko's templates use inline `<script>` and `<style>` blocks with per-request values, so hashes cannot work. **Fix:** a path-specific CSP. `identity.providerPage` replaces the header on provider routes only with `script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'`, keeping everything else same-origin plus `frame-ancestors 'none'`. Provider responses also get `X-Robots-Tag: noindex, nofollow`. `IDENTITY_TEMPLATE_DIR` could later supply templates with external scripts, which would let us drop `'unsafe-inline'`.

### 3. `BOOTSTRAP_CODE` is defined but never read

At this commit, `config.BootstrapCode` exists but no code uses it. Registration is open to anyone who can receive email. **Fix (site-side gate):** `POST /accounts/v1/register/*` is refused with 403 unless the browser carries a signed, 30-minute `da_enroll` cookie. That cookie comes from `POST /auth/enroll` with `IDENTITY_BOOTSTRAP_CODE`. The check is constant-time compare, rate-limited, with an Origin check. Existing passkeys sign in without it. With the code unset, enrollment is closed. Side effect: Portiko's sign-in page always says "a code is on its way" (deliberately, to avoid account enumeration), even when the gate refused. An unenrolled browser simply never gets a code. Upstream issue worth filing.

### 4. Portiko does not apply its own migrations

PLAN §5 assumed the provider migrates itself. It does not: `migrations` is an embedded FS for "the migrate CLI". **Fix:** `identity.Migrate` applies `portiko/migrations.FS` with golang-migrate to the identity database (its own `schema_migrations`). This runs on every `serve` and before each `wooddust identity` command. Because the database is separate, nothing clashes with the site's migrations.

### 5. The relying party discovers the provider over HTTP at startup

`rp.New` fetches `<issuer>/.well-known/openid-configuration` and the Portiko configuration before the server is listening, and the issuer is the site itself. **Fix:** the rp's `http.Client` uses an in-process `RoundTripper`. Requests for the issuer's host are served by `p.Handler()` directly (as `127.0.0.1`); others use the default transport. This also keeps the token exchange off Apache.

### 6. `return_to` versus `return`

`rp.RequireUser` redirects to `/auth/login?return_to=`, while the contract says `?return=`. `/auth/login` accepts both, and defaults to `/admin`. `rp` keeps the path local, so it cannot become an open redirect.

### 7. Sign-out leaves the provider session

`rp.LogoutHandler` only ends the site session, on purpose (single sign-on). With the provider dedicated to this site, the next `/auth/login` would complete without a passkey. **Fix:** `/auth/logout` first replays `POST /accounts/v1/logout` to the provider in-process with the browser's cookies, and copies the cleared SSO cookie back. Then it runs `rp`'s logout.

### 8. Extension points that cannot be used from outside

- `provider.Options.Mailer` is of type `internal/mailer.Mailer`, whose method signature uses the internal `PasskeyNotice`. An importer cannot implement it, so mail goes through Portiko's SMTP settings (`IDENTITY_SMTP_*`) or, unset, to the log.
- `Provider.DB()` returns `*internal/store.DB`, and its `UpsertClient` takes the internal `store.Client`. So client registration goes through `server.Main register-client` (wrapped as `register-site`) rather than happening at startup.
- `rp/session` ships only `MemoryStore`. We wrote the Postgres store.

### 9. Smaller notes

- The provider's per-IP token-endpoint budget (3600/hour) is shared by every in-process rp call, all of which come from `127.0.0.1`. That is far above one owner's use.
- `verify` (accepting Portiko access tokens on the API) is not wired. The admin API takes the session cookie or a `da_` API key (contract §7).
- The WebAuthn RP ID is the `BASE_URL` host, so passkeys do not survive a domain move (PLAN §10).
