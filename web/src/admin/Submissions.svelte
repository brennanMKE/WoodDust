<script lang="ts">
  // Submissions list: status/kind filters in the query string, cursor paging
  // (contract §7: newest first, ?status=&kind=&limit=&cursor=).
  import { app, replaceSearch } from '../lib/nav.svelte';
  import { formatDateTime } from '../lib/format';
  import { KINDS, kindLabel, statusLabel } from '../lib/submission';
  import { adminRequest, errorText, parseSubmissionPage, STATUSES, type Submission } from './api';

  const LIMIT = 25;
  // svelte-ignore state_referenced_locally
  const q = new URLSearchParams(app.search);
  let status = $state(q.get('status') ?? '');
  let kind = $state(q.get('kind') ?? '');
  let cursor = $state(q.get('cursor') ?? '');
  /** Cursors of the pages before this one, for "Newer". */
  let back: string[] = $state([]);
  let rows: Submission[] = $state([]);
  let next = $state<string | null>(null);
  let loading = $state(true);
  let error = $state('');
  let announce = $state('');

  $effect(() => {
    const params = new URLSearchParams();
    if (status) params.set('status', status);
    if (kind) params.set('kind', kind);
    if (cursor) params.set('cursor', cursor);
    replaceSearch(params.toString() ? `?${params}` : '');
    params.set('limit', String(LIMIT));
    const ctl = new AbortController();
    loading = true;
    error = '';
    adminRequest<unknown>('GET', `/api/v1/submissions?${params}`, undefined, ctl.signal)
      .then((body) => {
        const page = parseSubmissionPage(body);
        rows = page.submissions;
        next = page.next_cursor;
        loading = false;
        announce = `${rows.length} submission${rows.length === 1 ? '' : 's'} shown.`;
      })
      .catch((e) => {
        if (ctl.signal.aborted) return;
        loading = false;
        error = errorText(e);
      });
    return () => ctl.abort();
  });

  function filterChanged() {
    cursor = '';
    back = [];
  }

  function older() {
    if (!next) return;
    back = [...back, cursor];
    cursor = next;
  }

  function newer() {
    cursor = back[back.length - 1] ?? '';
    back = back.slice(0, -1);
  }

  const subject = (s: Submission) => s.target_slug ?? s.species_name ?? '—';
</script>

<h1>Submissions</h1>

<form class="filters" onsubmit={(e) => e.preventDefault()}>
  <div>
    <label for="f-status">Status</label>
    <select id="f-status" bind:value={status} onchange={filterChanged}>
      <option value="">Any</option>
      {#each STATUSES as s (s)}<option value={s}>{statusLabel(s)} ({s})</option>{/each}
    </select>
  </div>
  <div>
    <label for="f-kind">Kind</label>
    <select id="f-kind" bind:value={kind} onchange={filterChanged}>
      <option value="">Any</option>
      {#each KINDS as k (k.id)}<option value={k.id}>{k.label}</option>{/each}
    </select>
  </div>
</form>

<p class="visually-hidden" aria-live="polite">{announce}</p>

{#if error}
  <p class="error" role="alert">{error}</p>
{:else}
  <div class="table-wrap" aria-busy={loading}>
    <table>
      <caption class="visually-hidden">Submissions, newest first</caption>
      <thead>
        <tr>
          <th scope="col">Reference</th>
          <th scope="col">Kind</th>
          <th scope="col">Species</th>
          <th scope="col">Status</th>
          <th scope="col">Received</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as s (s.id)}
          <tr>
            <td><a class="mono" href="/admin/submissions/{s.id}">{s.reference}</a></td>
            <td>{kindLabel(s.kind)}</td>
            <td>{subject(s)}</td>
            <td><span class="pill status-{s.status}">{statusLabel(s.status)}</span></td>
            <td><time datetime={s.created_at}>{formatDateTime(s.created_at)}</time></td>
          </tr>
        {:else}
          <tr><td colspan="5" class="muted">{loading ? 'Loading…' : 'Nothing matches these filters.'}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
  <nav class="pager" aria-label="Pages">
    <button type="button" class="button" onclick={newer} disabled={!cursor || loading}>← Newer</button>
    <button type="button" class="button" onclick={older} disabled={!next || loading}>Older →</button>
  </nav>
{/if}

<style>
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin: 0.5rem 0 1.25rem;
  }

  .filters div {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .pager {
    display: flex;
    justify-content: space-between;
    margin-top: 1rem;
  }

  .pager .button[disabled] {
    opacity: 0.5;
    cursor: default;
  }
</style>
