<script lang="ts">
  // Settings (contract §7, session only): the ShortLinks key that mints a
  // link per plaque poster (docs/posters.md). The key is write-only here:
  // the server returns only its hint.
  import { onMount, tick } from 'svelte';
  import { ApiError } from '../lib/api';
  import { formatDateTime } from '../lib/format';
  import { adminRequest, errorText, getSettings, SHORTLINKS_STATUS, type ShortLinksView } from './api';

  let view = $state<ShortLinksView | null>(null);
  let loadError = $state('');
  let busy = $state<'' | 'test' | 'save' | 'clear'>('');
  let replacing = $state(false);
  let confirmingClear = $state(false);
  let value = $state('');
  let fieldError = $state('');
  let message = $state('');

  onMount(() => {
    const ctl = new AbortController();
    getSettings(ctl.signal)
      .then((r) => (view = r.shortlinks))
      .catch((e) => {
        if (!ctl.signal.aborted) loadError = errorText(e);
      });
    return () => {
      ctl.abort();
      value = '';
    };
  });

  async function act(kind: 'test' | 'save' | 'clear', req: () => Promise<{ shortlinks: ShortLinksView }>, done: string) {
    busy = kind;
    message = '';
    fieldError = '';
    try {
      view = (await req()).shortlinks;
      message = done;
      return true;
    } catch (e) {
      message = errorText(e);
      if (e instanceof ApiError && e.code === 'invalid') fieldError = message;
      return false;
    } finally {
      busy = '';
    }
  }

  async function test() {
    await act(
      'test',
      () => adminRequest('POST', '/api/v1/admin/settings/shortlinks/test'),
      'Tested.',
    );
    if (view?.status === 'failing') message = view.key.check_error ?? 'The test failed.';
    else if (view?.status === 'ok') message = 'Connection OK: the key works and the campaign exists.';
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!value.trim()) {
      fieldError = 'Paste the token from ShortLinks (it starts with slk_).';
      return;
    }
    const ok = await act(
      'save',
      () => adminRequest('PUT', '/api/v1/admin/settings/shortlinks_api_key', { value: value.trim() }),
      'Key saved. Run Test connection to check it.',
    );
    if (ok) {
      value = '';
      replacing = false;
      await tick();
      document.getElementById('test-btn')?.focus();
    }
  }

  async function clear() {
    const ok = await act(
      'clear',
      () => adminRequest('DELETE', '/api/v1/admin/settings/shortlinks_api_key'),
      'Key cleared. Plaque posters now use the stock QR code.',
    );
    if (ok) confirmingClear = false;
  }

  async function startReplace() {
    replacing = true;
    await tick();
    document.getElementById('sl-key')?.focus();
  }

  const statusClass = (s: string) => `pill sl-${s}`;
</script>

<h1>Settings</h1>

{#if loadError}
  <p class="error" role="alert">{loadError}</p>
{:else if !view}
  <p class="muted" aria-live="polite">Loading…</p>
{:else}
  <section class="card" aria-labelledby="sl-title">
    <div class="card-head">
      <h2 id="sl-title">ShortLinks</h2>
      <span class={statusClass(view.status)}>{SHORTLINKS_STATUS[view.status]}</span>
    </div>
    <p class="muted">
      Mints a QR code link for each poster with a “Posted by” plaque, in the <code>{view.campaign.hint}</code>
      campaign on {view.public_base.hint}. Without a working key, plaque posters use the stock link.
    </p>

    {#if !view.encryption_available}
      <div class="notice" role="alert">
        <p>
          <strong>SETTINGS_KEY is not set on this server,</strong> so the key can’t be stored. Run
          <code>make provision-&lt;env&gt;</code> to generate it, then restart the service.
        </p>
      </div>
    {/if}

    <dl>
      <dt>API key</dt>
      <dd>{#if view.key.set}<span class="mono">{view.key.hint}</span>{:else}<span class="muted">Not set</span>{/if}</dd>
      {#if view.key.set}
        <dt>Set</dt>
        <dd>{formatDateTime(view.key.set_at)} by <span class="mono">{view.key.set_by}</span></dd>
        <dt>Last check</dt>
        <dd>
          {#if view.key.checked_at}
            {formatDateTime(view.key.checked_at)}: {view.key.check_ok ? 'OK' : view.key.check_error}
          {:else}
            <span class="muted">Never</span>
          {/if}
        </dd>
      {/if}
      <dt>ShortLinks API</dt>
      <dd><span class="mono">{view.api_url.hint}</span>{#if view.api_url.is_default} <span class="muted">(default)</span>{/if}</dd>
    </dl>

    <p class="visually-hidden" aria-live="polite">{message}</p>
    {#if message}<p class={view.status === 'failing' && busy === '' && message === view.key.check_error ? 'error' : 'msg'}>{message}</p>{/if}

    <div class="actions">
      {#if view.key.set}
        <button type="button" class="button primary" id="test-btn" onclick={test} disabled={busy !== ''}>
          {busy === 'test' ? 'Testing…' : 'Test connection'}
        </button>
      {/if}
      {#if !replacing && view.encryption_available}
        <button type="button" class="button" onclick={startReplace} disabled={busy !== ''}>
          {view.key.set ? 'Replace key' : 'Set key'}
        </button>
      {/if}
      {#if view.key.set && !confirmingClear}
        <button type="button" class="button" onclick={() => (confirmingClear = true)} disabled={busy !== ''}>Clear</button>
      {/if}
    </div>

    {#if confirmingClear}
      <div class="confirm" role="group" aria-label="Confirm clearing the ShortLinks key">
        <span>Clear the key? Plaque posters fall back to the stock QR code.</span>
        <button type="button" class="button danger" onclick={clear} disabled={busy !== ''}>Clear key</button>
        <button type="button" class="button" onclick={() => (confirmingClear = false)}>Cancel</button>
      </div>
    {/if}

    {#if replacing}
      <form class="replace" onsubmit={save}>
        <label for="sl-key">New ShortLinks API token</label>
        <input
          id="sl-key"
          type="password"
          bind:value
          autocomplete="off"
          spellcheck="false"
          placeholder="slk_…"
          aria-invalid={fieldError ? true : undefined}
          aria-describedby="sl-key-hint{fieldError ? ' sl-key-error' : ''}"
        />
        <p class="hint" id="sl-key-hint">
          Create it on go.dustalmanac.com under Account › API tokens (not read-only). It’s stored encrypted and
          can’t be shown again; only its first 12 characters are displayed. From a terminal you can also use
          <code>scripts/set-setting.sh &lt;env&gt; shortlinks_api_key</code>.
        </p>
        {#if fieldError}<p class="error" id="sl-key-error">{fieldError}</p>{/if}
        <div class="actions">
          <button type="submit" class="button primary" disabled={busy !== ''}>{busy === 'save' ? 'Saving…' : 'Save key'}</button>
          <button type="button" class="button" onclick={() => ((replacing = false), (value = ''), (fieldError = ''))}>Cancel</button>
        </div>
      </form>
    {/if}
  </section>
{/if}

<style>
  .card {
    max-width: 44rem;
    padding: 1rem 1.15rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  .card-head {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .card-head h2 {
    margin: 0;
    font-size: 1.25rem;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.35rem 1rem;
    margin: 1rem 0;
  }

  dt {
    font-weight: 650;
  }

  dd {
    margin: 0;
    overflow-wrap: anywhere;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .confirm {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .replace {
    margin-top: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .replace input {
    font: inherit;
    font-family: var(--font-mono);
    max-width: 32rem;
    padding: 0.45rem 0.6rem;
    min-height: 40px;
    color: var(--text);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }

  .hint {
    font-size: 0.9rem;
    color: var(--muted);
    margin: 0;
  }

  .msg {
    margin: 0.5rem 0 0;
    font-weight: 600;
  }

  .button.danger {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--on-accent);
  }

  :global(.admin .pill.sl-ok) {
    background: var(--ev2-bg);
    color: var(--ev2-fg);
  }

  :global(.admin .pill.sl-failing),
  :global(.admin .pill.sl-not_configured) {
    background: var(--ev1-bg);
    color: var(--ev1-fg);
  }

  :global(.admin .pill.sl-untested) {
    background: var(--ev3-bg);
    color: var(--ev3-fg);
  }
</style>
