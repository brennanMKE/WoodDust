<script lang="ts">
  // /submit/status/{token} (contract §6): what happened to an anonymous submission.
  import { formatDateTime } from '../lib/format';
  import { STATUS_TEXT, kindLabel, statusLabel, timeline, type SubmissionStatus } from '../lib/submission';
  import type { SubmissionStatusData } from '../lib/types';

  let { data }: { data: SubmissionStatusData } = $props();

  const steps = $derived(timeline(data.status, data.kind));
  const now = $derived(
    STATUS_TEXT[(data.status === 'spam' ? 'new' : data.status) as SubmissionStatus] ?? {
      label: statusLabel(data.status),
      explain: '',
    },
  );
  const notes = $derived([...(data.notes ?? [])].sort((a, b) => a.at.localeCompare(b.at)));
</script>

<div class="wrap status">
  <p class="eyebrow">Submission {data.reference}</p>
  <h1>{now.label}</h1>
  {#if now.explain}<p class="lede">{now.explain}</p>{/if}

  <dl class="facts">
    <div><dt>Reference</dt><dd class="ref">{data.reference}</dd></div>
    <div><dt>Type</dt><dd>{kindLabel(data.kind)}</dd></div>
    <div><dt>Sent</dt><dd><time datetime={data.created_at}>{formatDateTime(data.created_at)}</time></dd></div>
    <div><dt>Last change</dt><dd><time datetime={data.updated_at}>{formatDateTime(data.updated_at)}</time></dd></div>
  </dl>

  <section aria-labelledby="timeline-title">
    <h2 id="timeline-title">Progress</h2>
    <ol class="timeline">
      {#each steps as s (s.status)}
        <li class={s.state} aria-current={s.state === 'current' ? 'step' : undefined}>
          <span class="dot" aria-hidden="true"></span>
          <span class="step-text">
            <span class="step-label">
              {s.label}
              <span class="visually-hidden">
                ({s.state === 'done' ? 'done' : s.state === 'current' ? 'current step' : 'not yet'})
              </span>
            </span>
            <span class="step-explain">{s.explain}</span>
          </span>
        </li>
      {/each}
    </ol>
  </section>

  <section aria-labelledby="notes-title">
    <h2 id="notes-title">Notes from the reviewer</h2>
    {#if notes.length}
      <ol class="notes">
        {#each notes as n, i (i)}
          <li>
            <p class="note-at"><time datetime={n.at}>{formatDateTime(n.at)}</time></p>
            <p class="note-text">{n.text}</p>
          </li>
        {/each}
      </ol>
    {:else}
      <p class="muted">No notes yet. Any questions we have for you will appear here.</p>
    {/if}
  </section>

  <div class="notice">
    <p>
      Keep this page’s address to check back. It’s the only link to your submission.
      {#if data.kind === 'reaction_report'}Reaction reports are never published. If you still have symptoms, please see a doctor.{/if}
    </p>
  </div>
  <p><a href="/">← Back to search</a></p>
</div>

<style>
  .status {
    max-width: 44rem;
    padding-top: 1.5rem;
  }

  .lede {
    color: var(--muted);
    font-size: 1.1rem;
  }

  .facts {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.75rem 1.5rem;
    margin: 1.5rem 0;
    padding: 0.9rem 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  .facts dt {
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--muted);
  }

  .facts dd {
    margin: 0;
    font-weight: 600;
  }

  .ref {
    font-family: var(--font-mono);
  }

  .timeline {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .timeline li {
    position: relative;
    display: flex;
    gap: 0.9rem;
    padding-bottom: 1.1rem;
  }

  .timeline li:not(:last-child)::before {
    content: '';
    position: absolute;
    left: 0.55rem;
    top: 1.4rem;
    bottom: 0;
    border-left: 2px solid var(--rule);
  }

  .timeline li.done:not(:last-child)::before {
    border-left-color: var(--blue);
  }

  .dot {
    flex: none;
    width: 1.2rem;
    height: 1.2rem;
    margin-top: 0.2rem;
    border-radius: 50%;
    border: 2px solid var(--border);
    background: var(--bg);
  }

  .done .dot {
    background: var(--blue);
    border-color: var(--blue);
  }

  .current .dot {
    border-color: var(--accent);
    box-shadow: inset 0 0 0 3px var(--bg);
    background: var(--accent);
  }

  .step-text {
    display: flex;
    flex-direction: column;
  }

  .step-label {
    font-weight: 650;
  }

  .upcoming .step-label {
    color: var(--muted);
    font-weight: 500;
  }

  .step-explain {
    font-size: 0.92rem;
    color: var(--muted);
  }

  .upcoming .step-explain {
    display: none;
  }

  .notes {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .notes li {
    padding: 0.75rem 1rem;
    border-left: 4px solid var(--wood);
    background: var(--surface);
    margin-bottom: 0.75rem;
  }

  .note-at {
    font-size: 0.85rem;
    color: var(--muted);
    margin: 0 0 0.25rem;
  }

  .note-text {
    margin: 0;
    white-space: pre-line;
  }
</style>
