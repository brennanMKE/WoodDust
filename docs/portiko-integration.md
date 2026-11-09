# Portiko integration

Dust Almanac embeds the Portiko identity provider in its own binary and uses Portiko's `rp` and `rp/session` packages for the site's session (PLAN §5). Only the owner signs in, and only for `/admin`. This page records how it is wired, what did not fit Portiko's assumptions at the pinned commit, and what we did about each mismatch without forking.

Pinned: `github.com/Kolonnade/Portiko v0.0.0-20261109082047-74783e489c6b` (`main` at 2026-11-09, `74783e4`). Code: `internal/identity`. Tests: `internal/identity/identity_test.go` drives the real provider with `descope/virtualwebauthn`. It covers the enrollment gate, Portiko refusing a wrong bootstrap code, passkey registration, sign-in, admin and non-admin users, the CSRF check, API key creation, and full sign-out.

## Wiring

| Piece | How |
|---|---|
| Provider config | Built in code (`identity.ProviderConfig`) from the site config, never from Portiko's env loader. `Registration` is set explicitly: `code` when `IDENTITY_BOOTSTRAP_CODE` is set (at least 16 characters), `closed` when it isn't. Issuer = `BASE_URL`; RP ID = its host; RP origins = `BASE_URL`; `ServiceName` = `SITE_NAME`; `TRUSTED_PROXIES` shared; `IDENTITY_*` for the database, crypto key, bootstrap code and SMTP; `DEV_INSECURE` for http://localhost |
| Database | Separate: `IDENTITY_DATABASE_URL` (`wooddust_identity`, `wooddust_identity_dev`, `wooddust_identity_test`) |
| Routes | `p.Handler()` is mounted only on `/oauth/`, `/accounts/`, `GET /.well-known/{openid-configuration,jwks.json,portiko-configuration,eureka-accounts-configuration}`, `GET /login`, `POST /login/select`, `GET /signin`, `GET /u/`. The site keeps `/`, `/health` and `/profile` |
| Site session | `rp.Client` with the Postgres `session.Store` in `internal/contrib/sessions.go` (`rp_sessions`; only the SHA-256 of the session id is stored). `/auth/login`, `/auth/callback`, `/auth/logout`, `/auth/backchannel-logout` |
| Authorization | Local `users` row keyed by subject. `OnSignIn` upserts it; the first subject ever becomes `is_admin` (advisory-locked). Others get 403 |
| Lifecycle | `go id.Sweep(ctx)` and `defer id.Close()` in `serve`; expired `rp_sessions` swept hourly |
| Operator commands | `wooddust identity <cmd>` maps `IDENTITY_*` onto Portiko's variable names, then calls `server.Main`. It also adds `migrate` and `register-site` (kept for manual use; the site registers itself on every start). `serve` is refused, because the provider is already embedded |
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

### 3. Registration and recovery are gated by the site as well

Until `c79d308` Portiko never read `BOOTSTRAP_CODE`. It now requires it for every new account (`REGISTRATION=code`), and WoodDust sets `closed` when no code is configured (Portiko P1/P1b). Portiko still lets an existing account recover by email in every mode, by design; for a single-owner admin site that would let whoever controls the owner's mailbox into `/admin`. So the site keeps its stricter gate in front. **Gate:** `POST /accounts/v1/register/*` is refused with 403 unless the browser carries a signed, 30-minute `da_enroll` cookie. That cookie comes from `POST /auth/enroll` with `IDENTITY_BOOTSTRAP_CODE`. The check is constant-time compare, rate-limited, with an Origin check. With the cookie, the gate fills Portiko's `bootstrap_code` field on `register/start`, so the code is typed once; the invitation-code field on Portiko's sign-in page can be left empty. Existing passkeys sign in without it. With the code unset, both the gate and Portiko refuse new accounts. Side effect: Portiko's sign-in page always says "a code is on its way" (deliberately, to avoid account enumeration), even when the gate refused. An unenrolled browser simply never gets a code.

### 4. Migrations (resolved upstream, P4)

`identity.Migrate` calls `provider.Migrate`: advisory-locked, one transaction per migration, refuses a dirty schema. It keeps golang-migrate's `schema_migrations (version, dirty)` table, so databases migrated before `74783e4` carry over. This runs on every `serve` and before each `wooddust identity` command. Because the database is separate, nothing clashes with the site's migrations.

### 5. The relying party discovers the provider over HTTP at startup

`rp.New` fetches `<issuer>/.well-known/openid-configuration` and the Portiko configuration before the server is listening, and the issuer is the site itself. **Fix:** the rp's `http.Client` uses an in-process `RoundTripper`. Requests for the issuer's host are served by `p.Handler()` directly (as `127.0.0.1`); others use the default transport. This also keeps the token exchange off Apache.

### 6. `return_to` versus `return`

`rp.RequireUser` redirects to `/auth/login?return_to=`, while the contract says `?return=`. `/auth/login` accepts both, and defaults to `/admin`. `rp` keeps the path local, so it cannot become an open redirect.

### 7. Sign-out leaves the provider session

`rp.LogoutHandler` only ends the site session, on purpose (single sign-on). With the provider dedicated to this site, the next `/auth/login` would complete without a passkey. **Fix:** `/auth/logout` first replays `POST /accounts/v1/logout` to the provider in-process with the browser's cookies, and copies the cleared SSO cookie back. Then it runs `rp`'s logout.

### 8. Extension points that cannot be used from outside

- **Mailer (P2, resolved upstream):** `provider.Mailer` can now be implemented outside Portiko. WoodDust still sends through Portiko's SMTP settings (`IDENTITY_SMTP_*`, SES) or, unset, to the log; an SES API mailer is optional.
- **Client registration (P3, resolved upstream):** `identity.New` calls `RegisterClient` with `identity.Client(cfg)` on every start: first-party, redirect `BASE_URL/auth/callback`, back-channel logout `BASE_URL/auth/backchannel-logout`. It replaces the declaration exactly but doesn't re-enable a client an operator switched off. `deploy.sh` no longer runs `register-site`.
- `rp/session` ships only `MemoryStore`. We wrote the Postgres store.

### 9. Smaller notes

- The provider's per-IP token-endpoint budget (3600/hour) is shared by every in-process rp call, all of which come from `127.0.0.1`. That is far above one owner's use.
- `verify` (accepting Portiko access tokens on the API) is not wired. The admin API takes the session cookie or a `da_` API key (contract §7).
- The WebAuthn RP ID is the `BASE_URL` host, so passkeys do not survive a domain move (PLAN §10).
