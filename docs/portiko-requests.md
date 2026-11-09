# Portiko change requests

This is the shared list of changes Dust Almanac needs from Portiko (`github.com/Kolonnade/Portiko`). Both the WoodDust session and the Portiko session use it to coordinate. The background is in [portiko-integration.md](portiko-integration.md).

- Pinned: `v0.0.0-20261106224428-4d9d3082a998` (`main`, 2026-11-06)
- Updated: 2026-11-08

## How to use this file

- **Portiko side:** when you start, fix, file, or decline an item, update its **Status** and **Portiko ref**. Use a commit hash, issue number, or PR number. Then message the WoodDust session.
- **WoodDust side:** when a fix lands, bump the pin and remove the workaround. Then mark the item `adopted` with the WoodDust commit.
- **Status values:** `open` → `in progress` → `fixed upstream` → `adopted`, or `declined` / `won't fix`. Give a reason for the last two.
- **Rules:** nobody pushes, tags, or touches a server (EC2/photon) without Brennan's explicit approval of that step.

## Summary

| # | Request | Priority | Status | Portiko ref | WoodDust ref |
|---|---|---|---|---|---|
| P1 | Enforce `BOOTSTRAP_CODE` on registration | **Security, high** | **adopted** (pin `74783e4`) | `c79d308`; `docs/registration.md` | explicit `Registration`; site gate kept for recovery, fills `bootstrap_code` |
| P1b | Option to close registration with no code set | **Security, high** | **adopted**: `REGISTRATION=open\|code\|closed`, `Config.Registration` | `c79d308` | `code` with a code, `closed` without |
| P2 | Exported, implementable `Mailer` | Medium | fixed upstream: `provider.Mailer`, `provider.PasskeyNotice` | `74783e4` | not used yet; SMTP (SES) still |
| P3 | Public client registration API | Medium | **adopted**: `(*Provider).RegisterClient(ctx, provider.Client)` | `74783e4` | `identity.Client`, on every start |
| P4 | Provider applies its own migrations | Medium | **adopted**: `provider.Migrate(ctx, dsn)` before `New`; `portiko migrate` | `74783e4` | `identity.Migrate` wraps it |
| P5 | Path-prefix option for `Handler()` | Medium | open | | cherry-picked routes |
| P6 | rp logout can end the provider SSO session | Medium | open | | in-process logout replay |
| P7 | CSP-friendly templates (no inline script/style) | Low | open | | path-specific CSP |
| P8 | Postgres `session.Store` in `rp/session` | Low | open | | `internal/contrib/sessions.go` |
| P9 | Document `return_to` and the embedding guide | Low | open | | `/auth/login` accepts both |
| P10 | Embedded `rp` discovery without HTTP | Low | open | | in-process `RoundTripper` |

---

## P1. Enforce `BOOTSTRAP_CODE` on registration (security)

**Problem.** `config.BootstrapCode` is loaded but never read. The `POST /accounts/v1/register/*` ceremony accepts anyone who can receive the email one-time code. So any deployment that relies on the bootstrap code to keep registration closed is actually open.

**Wanted.**
- When a bootstrap code is configured, registering a new subject requires it. Whether that means only the first subject, or every new one, is Portiko's call; document the choice.
- Use a constant-time comparison and a rate limit, and log refusals.
- Keep the anti-enumeration behavior ("a code is on its way"), but when the gate refuses, no code is sent or accepted.
- Add a config switch to close registration entirely, separate from the bootstrap code.
- Add tests that cover: code required, wrong code, correct code, existing passkeys still sign in, and the closed mode.
- Audit the other deployments that use Portiko for exposure today.

**WoodDust workaround.** `POST /auth/enroll` checks `IDENTITY_BOOTSTRAP_CODE` and sets a signed 30-minute `da_enroll` cookie. `POST /accounts/v1/register/*` is refused with a 403 unless the browser has that cookie. See [portiko-integration.md §3](portiko-integration.md).

**Done when.** WoodDust can delete the gate and still pass `TestSignInToAdmin` plus an "unenrolled email can't register" test.

**Portiko's fix (`c79d308`, pushed 2026-11-09):**
- When set, the code gates **every** new account, not only the first.
- The code is sent in a new `bootstrap_code` field on `POST /accounts/v1/register/start`.
- A wrong or missing code still returns `202 code_sent`, but no pending row is created and no mail goes out.
- The check is a constant-time compare.
- Recovery of an existing address never needs the code (Portiko `docs/recovery.md` §5).
- **An unset code means registration is open**, which is Portiko's documented intent.
- `Load()` rejects codes shorter than 16 characters. WoodDust builds `provider.Config` directly, so it skips that check and should enforce the minimum length itself.
- The sign-in page shows an "Invitation code (new accounts only)" field when a code is set. Custom `TEMPLATE_DIR` templates need that field added.

**What WoodDust does after adopting it:**
- Keep a site-side guard so that an unset `IDENTITY_BOOTSTRAP_CODE` still means **closed**: refuse `register/*` outright. Portiko treats unset as open, so deleting the env var (as `deployment.md` step 5.4 suggests) would otherwise open sign-up. P1b would remove the need for this guard.
- Replace the `da_enroll` cookie with Portiko's `bootstrap_code` field, or keep the cookie and have it inject the field.
- **Recovery is blocked behind the code on purpose.** WoodDust's gate also blocks email-only recovery. That conflicts with Portiko's recovery model, but it is deliberate for a single-owner admin site: someone who controls the owner's mailbox should not be able to take over `/admin`. The owner recovers with the bootstrap code from their password manager. Revisit this if WoodDust ever has more than one user.

## P1b. Option to close registration entirely

**Problem.** In Portiko, an unset `BOOTSTRAP_CODE` means registration is open, so the only way to close it is to set a code. A misconfigured or forgotten env var fails open.

**Wanted.** An explicit setting, for example `REGISTRATION=open|code|closed` (or `Options.Registration`), with a fail-closed default when a code is expected but missing. Keep `closed` separate from recovery, so existing accounts can still recover when the operator allows it.

## P2. Exported, implementable `Mailer`

**Problem.** `provider.Options.Mailer` has the type `internal/mailer.Mailer`. Its method signature uses the internal `PasskeyNotice` type, so no importer can implement it.

**Wanted.** An exported interface (for example `provider.Mailer`) whose argument types are all exported. Keep SMTP and log mailers as the built-in implementations.

**Why WoodDust cares.** It wants to send the one-time codes through SES or its own mail path without needing SMTP credentials.

## P3. Public client registration API

**Problem.** `Provider.DB()` returns `*internal/store.DB`, and `UpsertClient` takes the internal `store.Client`. An embedder can't register its own first-party client at startup.

**Wanted.** Something like `provider.Options.Clients []ClientConfig`, upserted at startup, or `(*Provider).RegisterClient(ctx, ClientConfig) error`.

**Done when.** WoodDust can drop the `register-site` operator step from `docs/deployment.md`.

## P4. Provider applies its own migrations

**Problem.** `migrations.FS` is only for the CLI, so embedders have to run golang-migrate themselves.

**Wanted.** One of these:
- `provider.Migrate(ctx, dsn)`;
- `Options.AutoMigrate bool`.

Document which one is expected.

## P5. Path-prefix option for `Handler()`

**Problem.** The router registers `GET /{$}`, `/health` and `/profile`, which collide with the host site's own routes. Embedders must mount a hand-picked list of paths. Templates and scripts also redirect to `/`. The list is in [portiko-integration.md §1](portiko-integration.md).

**Wanted.** `Options.BasePath` (for example `/id`). All routes, template links, redirects, and the `.well-known` issuer metadata should respect it. Alternatively, an option to skip registering the account-home, health and profile routes.

## P6. rp logout can end the provider SSO session

**Problem.** `rp.LogoutHandler` ends only the relying-party session, which is correct for multi-site SSO. With a dedicated, embedded provider, the next `/auth/login` then completes silently without a passkey.

**Wanted.** An option such as `rp.LogoutOptions{EndProviderSession: true}`, using RP-initiated logout (`end_session_endpoint`) or an in-process call.

**WoodDust workaround.** `/auth/logout` replays `POST /accounts/v1/logout` to the provider in-process and copies the cleared cookie back.

## P7. CSP-friendly templates

**Problem.** The templates use inline `<script>` and `<style>` with per-request values. That forces `'unsafe-inline'` on the provider paths.

**Wanted.** Either external JS and CSS files, with per-request values passed through `data-*` attributes or a JSON `<script type="application/json">`, or per-request nonces exposed to the host so it can set its CSP.

## P8. Postgres `session.Store` in `rp/session`

**Problem.** Only `MemoryStore` ships, so every site writes its own store.

**Wanted.** A pgx-based store that keeps only a hash of the session id and has a sweep. WoodDust's `internal/contrib/sessions.go` (table `rp_sessions`, SHA-256 ids) can serve as the starting point.

## P9. Documentation

- Document the `rp.RequireUser` redirect parameter `return_to`.
- Write an "embedding the provider in your site" guide covering the items above:
  - which routes it needs;
  - migrations;
  - the CSP;
  - discovery when the issuer is the site itself;
  - the shared per-IP token budget when every rp call comes from `127.0.0.1`.

## P10. Embedded `rp` discovery without HTTP

**Problem.** `rp.New` fetches `<issuer>/.well-known/*` over HTTP before the host server is listening. When the issuer is the same process, that fails.

**Wanted.** One of these:
- `rp.Config.Provider *provider.Provider` (or an `http.Handler`) to use in-process;
- lazy discovery on first use.

**WoodDust workaround.** A custom `RoundTripper` serves requests for the issuer's host through `p.Handler()`.
