<script lang="ts">
  // One submission: full content, status + notes (PATCH), event history and
  // the export draft (contract §7).
  import { onMount, tick } from 'svelte';
  import { doiHref, pmidHref } from '../lib/citations';
  import { formatDateTime } from '../lib/format';
  import { hazardName } from '../lib/labels';
  import { kindLabel, statusLabel, warningTexts } from '../lib/submission';
  import CopyButton from '../components/CopyButton.svelte';
  import {
    adminRequest,
    errorText,
    parseSubmissionDetail,
    STATUSES,
    type Submission,
    type SubmissionEvent,
  } from './api';

  let { id }: { id: string } = $props();

  let sub = $state<Submission | null>(null);
  let events: SubmissionEvent[] = $state([]);
  let loadError = $state('');

  let newStatus = $state('');
  let noteText = $state('');
  let noteVisible = $state(false);
  let saving = $state(false);
  let saveError = $state('');
  let saveMessage = $state('');

  let exportJSON = $state('');
  let exportError = $state('');
  let exporting = $state(false);
  let exportEl: HTMLElement | undefined = $state();

  async function load() {
    try {
      const d = parseSubmissionDetail(await adminRequest<unknown>('GET', `/api/v1/submissions/${encodeURIComponent(id)}`));
      sub = d.submission;
      events = d.events;
      newStatus = d.submission.status;
    } catch (e) {
      loadError = errorText(e);
    }
  }

  onMount(() => {
    void load();
  });

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!sub) return;
    const body: { status?: string; note?: { text: string; visible_to_submitter: boolean } } = {};
    if (newStatus && newStatus !== sub.status) body.status = newStatus;
    if (noteText.trim()) body.note = { text: noteText.trim(), visible_to_submitter: noteVisible };
    if (!body.status && !body.note) {
      saveError = 'Change the status or write a note first.';
      return;
    }
    saving = true;
    saveError = '';
    saveMessage = '';
    try {
      await adminRequest('PATCH', `/api/v1/submissions/${encodeURIComponent(id)}`, body);
      noteText = '';
      noteVisible = false;
      await load();
      saveMessage = body.status ? `Saved. Status is now ${statusLabel(sub.status)}.` : 'Note added.';
    } catch (err) {
      saveError = errorText(err);
    } finally {
      saving = false;
    }
  }

  async function showExport() {
    exporting = true;
    exportError = '';
    try {
      const draft = await adminRequest<unknown>('GET', `/api/v1/submissions/${encodeURIComponent(id)}/export`);
      exportJSON = JSON.stringify(draft, null, 2);
      await tick();
      exportEl?.focus();
    } catch (err) {
      exportError = errorText(err);
    } finally {
      exporting = false;
    }
  }

  function eventSummary(ev: SubmissionEvent): string {
    const to = ev.to_status ?? ev.status;
    const what = ev.type ?? ev.action ?? (to ? 'status' : ev.note || ev.text ? 'note' : 'event');
    if (to) return `${ev.from_status ? `${statusLabel(ev.from_status)} → ` : 'Status → '}${statusLabel(to)}`;
    return what.replace(/_/g, ' ');
  }

  function eventNote(ev: SubmissionEvent): { text: string; visible: boolean } | null {
    if (typeof ev.note === 'string') return { text: ev.note, visible: Boolean(ev.visible_to_submitter) };
    if (ev.note && typeof ev.note === 'object') return { text: ev.note.text, visible: Boolean(ev.note.visible_to_submitter) };
    if (ev.text) return { text: ev.text, visible: Boolean(ev.visible_to_submitter) };
    return null;
  }

  const canExport = $derived(sub ? ['accepted', 'exported', 'merged'].includes(sub.status) : false);
  const warnings = $derived(sub ? warningTexts(sub.warnings) : []);
</script>

<p class="back"><a href="/admin/submissions">← All submissions</a></p>
<h1>Submission {sub?.reference ?? `#${id}`}</h1>

{#if loadError}
  <p class="error" role="alert">{loadError}</p>
{:else if !sub}
  <p class="muted" aria-live="polite">Loading…</p>
{:else}
  <div class="layout">
    <div class="main">
      <dl class="facts">
        <div><dt>Status</dt><dd><span class="pill status-{sub.status}">{statusLabel(sub.status)}</span></dd></div>
        <div><dt>Kind</dt><dd>{kindLabel(sub.kind)}</dd></div>
        <div>
          <dt>Species</dt>
          <dd>
            {#if sub.target_slug}<a href="/wood/{sub.target_slug}">{sub.target_slug}</a>{/if}
            {#if sub.species_name}{sub.target_slug ? ' · ' : ''}{sub.species_name}{/if}
            {#if sub.botanical_name}<span class="botanical"> ({sub.botanical_name})</span>{/if}
            {#if !sub.target_slug && !sub.species_name}—{/if}
          </dd>
        </div>
        <div><dt>Hazard</dt><dd>{sub.hazard_type ? hazardName(sub.hazard_type) : '—'}</dd></div>
        <div><dt>Received</dt><dd><time datetime={sub.created_at}>{formatDateTime(sub.created_at)}</time></dd></div>
        {#if sub.updated_at}<div><dt>Updated</dt><dd><time datetime={sub.updated_at}>{formatDateTime(sub.updated_at)}</time></dd></div>{/if}
        <div><dt>Email</dt><dd>{#if sub.email}<a href="mailto:{sub.email}">{sub.email}</a>{:else}<span class="muted">none</span>{/if}</dd></div>
        <div><dt>IP hash</dt><dd class="mono hash">{sub.ip_hash ?? '—'}</dd></div>
      </dl>

      <section aria-labelledby="details-title">
        <h2 id="details-title">Details</h2>
        <p class="details">{sub.details}</p>
      </section>

      <section aria-labelledby="cites-title">
        <h2 id="cites-title">Citations</h2>
        {#if sub.citations?.length}
          <ol class="cites">
            {#each sub.citations as c, i (i)}
              <li>
                {#if c.doi}<a href={doiHref(c.doi)} rel="external noopener" target="_blank">DOI {c.doi}</a>{/if}
                {#if c.pmid}<a href={pmidHref(c.pmid)} rel="external noopener" target="_blank">PMID {c.pmid}</a>{/if}
                {#if c.url}<a href={c.url} rel="external noopener nofollow" target="_blank" class="url">{c.url}</a>{/if}
                {#if c.text}<span class="cite-text">{c.text}</span>{/if}
              </li>
            {/each}
          </ol>
        {:else}
          <p class="muted">None given.</p>
        {/if}
        {#if warnings.length}
          <ul class="warn">{#each warnings as w, i (i)}<li>{w}</li>{/each}</ul>
        {/if}
      </section>

      <section aria-labelledby="history-title">
        <h2 id="history-title">History</h2>
        {#if events.length}
          <ol class="events">
            {#each events as ev, i (i)}
              {@const note = eventNote(ev)}
              <li>
                <p class="ev-head">
                  <time datetime={ev.at}>{formatDateTime(ev.at)}</time>
                  {#if ev.actor}<span class="muted"> · {ev.actor}</span>{/if}
                </p>
                <p class="ev-what">{eventSummary(ev)}</p>
                {#if note}
                  <p class="ev-note">
                    {note.text}
                    <span class="pill">{note.visible ? 'Visible to submitter' : 'Private'}</span>
                  </p>
                {/if}
              </li>
            {/each}
          </ol>
        {:else}
          <p class="muted">No events yet.</p>
        {/if}
      </section>
    </div>

    <aside class="side">
      <form class="panel" onsubmit={save} aria-labelledby="update-title">
        <h2 id="update-title">Update</h2>
        <div class="field">
          <label for="new-status">Status</label>
          <select id="new-status" bind:value={newStatus}>
            {#each STATUSES as s (s)}<option value={s}>{statusLabel(s)} ({s})</option>{/each}
          </select>
        </div>
        <div class="field">
          <label for="note">Note <span class="muted">(optional)</span></label>
          <textarea id="note" rows="4" bind:value={noteText}></textarea>
        </div>
        <label class="check">
          <input type="checkbox" bind:checked={noteVisible} aria-describedby="visible-hint" />
          Visible to submitter
        </label>
        <p class="hint" id="visible-hint">Visible notes appear on the submitter’s status page. Don’t include their email.</p>
        <button type="submit" class="button primary" disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
        <p class="error" role="alert">{saveError}</p>
        <p role="status">{saveMessage}</p>
      </form>

      <section class="panel" aria-labelledby="export-title">
        <h2 id="export-title">Export</h2>
        {#if canExport}
          <p class="hint">A draft patch for <code>data/species/&lt;slug&gt;.json</code>. Review it, then commit it by hand.</p>
          <button type="button" class="button" onclick={showExport} disabled={exporting}>
            {exporting ? 'Loading…' : exportJSON ? 'Refresh draft' : 'Show draft JSON'}
          </button>
        {:else}
          <p class="hint">Available once the submission is accepted.</p>
        {/if}
        {#if exportError}<p class="error" role="alert">{exportError}</p>{/if}
      </section>
    </aside>
  </div>

  {#if exportJSON}
    <section class="export" aria-labelledby="draft-title">
      <div class="export-head">
        <h2 id="draft-title">Draft JSON</h2>
        <CopyButton text={exportJSON} label="Copy JSON" what="JSON" selectTarget={exportEl} />
      </div>
      <pre bind:this={exportEl} tabindex="-1" aria-label="Draft JSON"><code>{exportJSON}</code></pre>
    </section>
  {/if}
{/if}

<style>
  .back {
    margin: 0 0 0.25rem;
  }

  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 20rem;
    gap: 2rem;
    align-items: start;
  }

  @media (max-width: 52rem) {
    .layout {
      grid-template-columns: 1fr;
    }
  }

  .facts {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: 0.75rem 1.5rem;
    margin: 0.5rem 0 1rem;
  }

  .facts dt {
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--muted);
  }

  .facts dd {
    margin: 0;
    overflow-wrap: anywhere;
  }

  .hash {
    font-size: 0.8rem;
  }

  .details {
    white-space: pre-wrap;
    padding: 0.75rem 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius-sm);
  }

  .cites {
    padding-left: 1.4rem;
  }

  .cites li {
    margin-bottom: 0.4rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 0.75rem;
  }

  .url {
    overflow-wrap: anywhere;
  }

  .cite-text {
    color: var(--muted);
    flex-basis: 100%;
  }

  .warn {
    padding: 0.5rem 1rem 0.5rem 2rem;
    background: var(--ev3-bg);
    color: var(--ev3-fg);
    border-radius: var(--radius-sm);
  }

  .events {
    list-style: none;
    padding: 0;
    margin: 0;
    border-left: 2px solid var(--rule);
  }

  .events li {
    padding: 0 0 0.9rem 1rem;
  }

  .events p {
    margin: 0;
  }

  .ev-head {
    font-size: 0.85rem;
    color: var(--muted);
  }

  .ev-what {
    font-weight: 600;
  }

  .ev-note {
    white-space: pre-line;
  }

  .side {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .panel {
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  .panel h2 {
    margin: 0 0 0.75rem;
    font-size: 1.2rem;
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
  }

  .check input {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--blue);
  }

  .hint {
    font-size: 0.88rem;
    color: var(--muted);
    margin: 0.3rem 0 0.75rem;
  }

  .panel p:empty {
    display: none;
  }

  .export {
    margin-top: 2rem;
  }

  .export-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1rem;
  }

  .export-head h2 {
    margin: 0;
  }

  pre {
    margin-top: 0.75rem;
    padding: 1rem;
    max-height: 32rem;
    overflow: auto;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius-sm);
    font-family: var(--font-mono);
    font-size: 0.88rem;
  }
</style>
