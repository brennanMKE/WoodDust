<script lang="ts">
  import { onMount } from 'svelte';
  import { formatDateTime } from '../lib/format';
  import { kindLabel, statusLabel } from '../lib/submission';
  import { adminRequest, errorText, getSettings, SHORTLINKS_STATUS, STATUSES, type ShortLinksView, type Stats } from './api';

  let stats = $state<Stats | null>(null);
  let error = $state('');
  let shortlinks = $state<ShortLinksView | null>(null);

  onMount(() => {
    const ctl = new AbortController();
    // The banner is a nudge; if settings can't load, the Settings tab says why.
    getSettings(ctl.signal)
      .then((s) => (shortlinks = s.shortlinks))
      .catch(() => undefined);
    adminRequest<Stats>('GET', '/api/v1/stats', undefined, ctl.signal)
      .then((s) => (stats = s))
      .catch((e) => {
        if (!ctl.signal.aborted) error = errorText(e);
      });
    return () => ctl.abort();
  });

  const statusRows = $derived(
    stats
      ? [
          ...STATUSES.filter((s) => s in stats!.by_status),
          ...Object.keys(stats.by_status).filter((s) => !(STATUSES as readonly string[]).includes(s)),
        ].map((s) => [s, stats!.by_status[s]] as const)
      : [],
  );
  const kindRows = $derived(stats ? Object.entries(stats.by_kind) : []);
</script>

<h1>Dashboard</h1>

{#if shortlinks && shortlinks.status !== 'ok'}
  <div class="notice banner" role="status">
    <p>
      <strong>Poster links: {SHORTLINKS_STATUS[shortlinks.status]}.</strong>
      {#if shortlinks.status === 'failing'}{shortlinks.key.check_error}{:else if shortlinks.status === 'untested'}Run Test connection to check the ShortLinks key.{:else}Plaque posters use the stock QR code until the ShortLinks key is set.{/if}
      <a href="/admin/settings">Open Settings</a>.
    </p>
  </div>
{/if}

{#if error}
  <p class="error" role="alert">{error}</p>
{:else if !stats}
  <p class="muted" aria-live="polite">Loading counts…</p>
{:else}
  <ul class="tiles">
    <li>
      <a href="/admin/submissions">
        <span class="n">{stats.open}</span><span class="t">Open</span>
      </a>
    </li>
    <li><span class="n">{stats.new_last_24h}</span><span class="t">New in the last 24 hours</span></li>
    <li><span class="n">{stats.spam_last_24h}</span><span class="t">Spam caught in the last 24 hours</span></li>
    <li>
      <span class="n small">{stats.newest_at ? formatDateTime(stats.newest_at) : '—'}</span>
      <span class="t">Newest submission</span>
    </li>
  </ul>

  <div class="cols">
    <section aria-labelledby="by-status">
      <h2 id="by-status">By status</h2>
      <table>
        <thead><tr><th scope="col">Status</th><th scope="col" class="num">Count</th></tr></thead>
        <tbody>
          {#each statusRows as [s, n] (s)}
            <tr>
              <td><a href="/admin/submissions?status={encodeURIComponent(s)}">{statusLabel(s)}</a> <span class="muted mono">{s}</span></td>
              <td class="num">{n}</td>
            </tr>
          {:else}
            <tr><td colspan="2" class="muted">No submissions yet.</td></tr>
          {/each}
        </tbody>
      </table>
    </section>
    <section aria-labelledby="by-kind">
      <h2 id="by-kind">By kind</h2>
      <table>
        <thead><tr><th scope="col">Kind</th><th scope="col" class="num">Count</th></tr></thead>
        <tbody>
          {#each kindRows as [k, n] (k)}
            <tr>
              <td><a href="/admin/submissions?kind={encodeURIComponent(k)}">{kindLabel(k)}</a></td>
              <td class="num">{n}</td>
            </tr>
          {:else}
            <tr><td colspan="2" class="muted">No submissions yet.</td></tr>
          {/each}
        </tbody>
      </table>
    </section>
  </div>
{/if}

<style>
  .banner {
    margin: 0.5rem 0 1rem;
  }

  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
    gap: 0.75rem;
    list-style: none;
    margin: 1rem 0 2rem;
    padding: 0;
  }

  .tiles li,
  .tiles a {
    display: flex;
    flex-direction: column;
  }

  .tiles li {
    padding: 0.9rem 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-top: 4px solid var(--accent);
    border-radius: var(--radius-sm);
  }

  .tiles a {
    color: inherit;
    text-decoration: none;
  }

  .tiles a:hover .t {
    text-decoration: underline;
  }

  .n {
    font-family: var(--font-serif);
    font-size: 2.2rem;
    font-weight: 700;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
  }

  .n.small {
    font-size: 1.1rem;
    padding: 0.55rem 0;
  }

  .t {
    color: var(--muted);
    font-size: 0.92rem;
  }

  .cols {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
    gap: 0 2rem;
  }

  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
</style>
