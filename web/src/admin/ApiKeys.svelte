<script lang="ts">
  // API keys (contract §7, session only). A new key is shown exactly once.
  import { onMount, tick } from 'svelte';
  import { formatDateTime } from '../lib/format';
  import CopyButton from '../components/CopyButton.svelte';
  import { adminRequest, errorText } from './api';
  import { OneTimeSecret, SCOPES, keyState, maskedKey, parseKeyList, toListEntry, type ApiKey } from './keys';

  let keys: ApiKey[] = $state([]);
  let loading = $state(true);
  let listError = $state('');

  let name = $state('');
  let scopes: string[] = $state(['submissions:read']);
  let expires = $state('');
  let creating = $state(false);
  let createError = $state('');

  const secret = new OneTimeSecret();
  /** Mirrors `secret` for rendering; cleared together with it. */
  let shown = $state<{ key: string; prefix: string } | null>(null);
  let revealEl: HTMLElement | undefined = $state();
  let keyEl: HTMLElement | undefined = $state();

  let confirming = $state<string | null>(null);
  let revoking = $state(false);
  let announce = $state('');

  async function loadKeys() {
    try {
      keys = parseKeyList(await adminRequest<unknown>('GET', '/api/v1/admin/api-keys'));
      listError = '';
    } catch (e) {
      listError = errorText(e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadKeys();
    // Never leave the secret in memory after leaving the page.
    return () => secret.dismiss();
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    createError = '';
    if (!name.trim()) {
      createError = 'Give the key a name, such as what will use it.';
      document.getElementById('key-name')?.focus();
      return;
    }
    if (!scopes.length) {
      createError = 'Choose at least one scope.';
      return;
    }
    creating = true;
    try {
      const body: { name: string; scopes: string[]; expires_at?: string } = { name: name.trim(), scopes: [...scopes] };
      // A date input gives a day; the key expires at the end of that day (UTC).
      if (expires) body.expires_at = `${expires}T23:59:59Z`;
      const created = await adminRequest<ApiKey & { key: string }>('POST', '/api/v1/admin/api-keys', body);
      secret.reveal(created.key, created.prefix);
      shown = secret.value ? { key: secret.value, prefix: created.prefix } : null;
      keys = [toListEntry(created), ...keys.filter((k) => k.prefix !== created.prefix)];
      name = '';
      expires = '';
      scopes = ['submissions:read'];
      await tick();
      revealEl?.focus();
    } catch (err) {
      createError = errorText(err);
    } finally {
      creating = false;
    }
  }

  async function dismissSecret() {
    secret.dismiss();
    shown = null;
    announce = 'The key is hidden and can’t be shown again.';
    await tick();
    document.getElementById('key-name')?.focus();
  }

  async function askRevoke(prefix: string) {
    confirming = prefix;
    await tick();
    document.getElementById(`confirm-${prefix}`)?.focus();
  }

  async function cancelRevoke(prefix: string) {
    confirming = null;
    await tick();
    document.getElementById(`revoke-${prefix}`)?.focus();
  }

  async function revoke(k: ApiKey) {
    revoking = true;
    try {
      await adminRequest('DELETE', `/api/v1/admin/api-keys/${encodeURIComponent(k.prefix)}`);
      confirming = null;
      announce = `Key “${k.name}” revoked.`;
      await loadKeys();
    } catch (err) {
      announce = `Couldn’t revoke “${k.name}”: ${errorText(err)}`;
    } finally {
      revoking = false;
    }
  }

  const today = new Date().toISOString().slice(0, 10);
</script>

<h1>API keys</h1>
<p class="muted">
  Keys let scripts read or update submissions without signing in. Send them as <code>Authorization: Bearer da_…</code>.
  Only a hash of each key is stored.
</p>

{#if shown}
  <section class="reveal" aria-labelledby="reveal-title" tabindex="-1" bind:this={revealEl}>
    <h2 id="reveal-title">Copy your new key now</h2>
    <p>
      <strong>This is the only time it will be shown.</strong> We keep only a hash, so if you lose it you’ll need to
      create a new key and revoke this one.
    </p>
    <p class="secret"><code bind:this={keyEl}>{shown.key}</code></p>
    <p class="reveal-actions">
      <CopyButton text={shown.key} label="Copy key" what="Key" selectTarget={keyEl} />
      <button type="button" class="button" onclick={dismissSecret}>I’ve saved it, hide the key</button>
    </p>
  </section>
{/if}

<form class="create" onsubmit={create} aria-labelledby="create-title">
  <h2 id="create-title">Create a key</h2>
  <div class="field">
    <label for="key-name">Name</label>
    <input id="key-name" type="text" bind:value={name} maxlength="80" placeholder="e.g. triage script on my laptop" autocomplete="off" />
  </div>
  <fieldset>
    <legend>Scopes</legend>
    {#each SCOPES as s (s.id)}
      <label class="check">
        <input type="checkbox" value={s.id} bind:group={scopes} />
        <span><code>{s.id}</code> <span class="muted">{s.label}</span></span>
      </label>
    {/each}
  </fieldset>
  <div class="field">
    <label for="key-expires">Expires <span class="muted">(optional)</span></label>
    <input id="key-expires" type="date" min={today} bind:value={expires} aria-describedby="expires-hint" />
    <p class="hint" id="expires-hint">Leave empty for a key that doesn’t expire. You can revoke any key at any time.</p>
  </div>
  {#if createError}<p class="error" role="alert">{createError}</p>{/if}
  <button type="submit" class="button primary" disabled={creating}>{creating ? 'Creating…' : 'Create key'}</button>
</form>

<section aria-labelledby="list-title">
  <h2 id="list-title">Keys</h2>
  <p class="visually-hidden" aria-live="polite">{announce}</p>
  {#if listError}
    <p class="error" role="alert">{listError}</p>
  {:else}
    <div class="table-wrap" aria-busy={loading}>
      <table>
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col">Key</th>
            <th scope="col">Scopes</th>
            <th scope="col">Created</th>
            <th scope="col">Last used</th>
            <th scope="col">Expires</th>
            <th scope="col"><span class="visually-hidden">Actions</span></th>
          </tr>
        </thead>
        <tbody>
          {#each keys as k (k.prefix)}
            {@const st = keyState(k)}
            <tr class:inactive={st !== 'active'}>
              <td>{k.name}</td>
              <td class="mono">{maskedKey(k.prefix)}</td>
              <td>{#each k.scopes as s (s)}<code class="scope">{s}</code> {/each}</td>
              <td>{formatDateTime(k.created_at)}</td>
              <td>{k.last_used_at ? formatDateTime(k.last_used_at) : 'Never'}</td>
              <td>{k.expires_at ? formatDateTime(k.expires_at) : 'Never'}</td>
              <td class="act">
                {#if st === 'revoked'}
                  <span class="pill">Revoked</span>
                {:else if confirming === k.prefix}
                  <span class="confirm" role="group" aria-label="Confirm revoking {k.name}">
                    <span>Revoke “{k.name}”? Anything using it stops working.</span>
                    <button type="button" class="button danger" id="confirm-{k.prefix}" onclick={() => revoke(k)} disabled={revoking}>
                      Revoke
                    </button>
                    <button type="button" class="button" onclick={() => cancelRevoke(k.prefix)}>Cancel</button>
                  </span>
                {:else}
                  {#if st === 'expired'}<span class="pill">Expired</span>{/if}
                  <button type="button" class="button" id="revoke-{k.prefix}" onclick={() => askRevoke(k.prefix)}>
                    Revoke<span class="visually-hidden"> {k.name}</span>
                  </button>
                {/if}
              </td>
            </tr>
          {:else}
            <tr><td colspan="7" class="muted">{loading ? 'Loading…' : 'No keys yet.'}</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>

<style>
  .reveal {
    margin: 1rem 0 1.5rem;
    padding: 1rem 1.15rem;
    border: 2px solid var(--accent);
    border-left-width: 8px;
    border-radius: var(--radius-sm);
    background: var(--surface);
  }

  .reveal h2 {
    margin: 0 0 0.5rem;
    color: var(--accent);
  }

  .secret code {
    display: block;
    padding: 0.6rem 0.8rem;
    background: var(--surface-2);
    border-radius: var(--radius-sm);
    font-family: var(--font-mono);
    overflow-wrap: anywhere;
    user-select: all;
  }

  .reveal-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin: 0;
  }

  .create {
    max-width: 36rem;
    margin: 1rem 0 2rem;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  .create h2 {
    margin: 0 0 0.75rem;
    font-size: 1.2rem;
  }

  fieldset {
    border: 0;
    padding: 0;
    margin: 0 0 0.75rem;
  }

  legend {
    font-weight: 600;
    margin-bottom: 0.25rem;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    margin-bottom: 0.75rem;
  }

  .check {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    font-weight: 400;
    padding: 0.2rem 0;
  }

  .check input {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--blue);
  }

  .hint {
    font-size: 0.88rem;
    color: var(--muted);
    margin: 0;
  }

  .scope {
    font-size: 0.82rem;
    white-space: nowrap;
  }

  .inactive td {
    color: var(--muted);
  }

  .act {
    white-space: nowrap;
  }

  .confirm {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    white-space: normal;
  }

  .button.danger {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--on-accent);
  }

  .act .button {
    min-height: 36px;
    padding: 0.25rem 0.75rem;
  }
</style>
