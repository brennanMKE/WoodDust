<script lang="ts">
  // Owner-only admin (contract §7). Loaded as its own chunk from App.svelte so
  // none of this reaches the public bundle.
  import { onMount } from 'svelte';
  import { app } from '../lib/nav.svelte';
  import type { MeResponse } from '../lib/types';
  import Dashboard from './Dashboard.svelte';
  import Submissions from './Submissions.svelte';
  import SubmissionDetail from './SubmissionDetail.svelte';
  import ApiKeys from './ApiKeys.svelte';

  let { data, section, id }: { data: MeResponse | null; section: string; id?: string } = $props();

  const user = $derived(data?.user);
  const tabs = [
    { href: '/admin', label: 'Dashboard', section: 'dashboard' },
    { href: '/admin/submissions', label: 'Submissions', section: 'submissions' },
    { href: '/admin/api-keys', label: 'API keys', section: 'api-keys' },
  ];
  const activeTab = $derived(section === 'submission' ? 'submissions' : section);

  // The app focuses the new <h1> after navigation; this chunk may mount a beat
  // later, so do it here too if focus was left behind.
  onMount(() => {
    if (app.navCount === 0) return;
    const active = document.activeElement;
    if (!active || active === document.body || active.id === 'main') {
      const h1 = document.querySelector<HTMLElement>('#main h1');
      if (h1) {
        h1.setAttribute('tabindex', '-1');
        h1.focus({ preventScroll: true });
      }
    }
  });
</script>

<div class="wrap admin">
  <div class="bar">
    <nav aria-label="Admin">
      <ul>
        {#each tabs as t (t.href)}
          <li><a href={t.href} aria-current={activeTab === t.section ? 'page' : undefined}>{t.label}</a></li>
        {/each}
      </ul>
    </nav>
    <p class="who">
      {#if user}Signed in as <strong>{user.display_name}</strong> ·{/if}
      <a href="/auth/logout" rel="external">Sign out</a>
    </p>
  </div>

  {#if user && !user.is_admin}
    <h1>Owner only</h1>
    <div class="notice" role="alert">
      <p>You’re signed in, but this area is only for the site owner. <a href="/auth/logout" rel="external">Sign out</a>.</p>
    </div>
  {:else if section === 'dashboard'}
    <Dashboard />
  {:else if section === 'submissions'}
    <Submissions />
  {:else if section === 'submission' && id}
    <SubmissionDetail id={id} />
  {:else if section === 'api-keys'}
    <ApiKeys />
  {/if}
</div>

<style>
  .admin {
    padding-top: 1rem;
  }

  .bar {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem 1.5rem;
    padding-bottom: 0.6rem;
    margin-bottom: 1.25rem;
    border-bottom: 1px solid var(--rule);
  }

  nav ul {
    display: flex;
    gap: 0.25rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  nav a {
    display: inline-block;
    padding: 0.45rem 0.8rem;
    border-radius: var(--radius-sm);
    text-decoration: none;
    font-weight: 600;
  }

  nav a[aria-current='page'] {
    background: var(--blue);
    color: var(--on-blue);
  }

  .who {
    margin: 0;
    font-size: 0.93rem;
    color: var(--muted);
  }

  /* Shared admin styles for the sub-views. */
  .admin :global(table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.95rem;
  }

  .admin :global(th),
  .admin :global(td) {
    text-align: left;
    padding: 0.5rem 0.6rem;
    border-bottom: 1px solid var(--rule);
    vertical-align: top;
  }

  .admin :global(th) {
    font-size: 0.82rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--muted);
  }

  .admin :global(.table-wrap) {
    overflow-x: auto;
  }

  .admin :global(.pill) {
    display: inline-block;
    padding: 0.05rem 0.5rem;
    border-radius: 999px;
    font-size: 0.82rem;
    font-weight: 650;
    background: var(--surface-2);
    border: 1px solid var(--rule);
    white-space: nowrap;
  }

  .admin :global(.pill.status-new),
  .admin :global(.pill.status-needs_info) {
    background: var(--ev1-bg);
    color: var(--ev1-fg);
  }

  .admin :global(.pill.status-accepted),
  .admin :global(.pill.status-exported),
  .admin :global(.pill.status-merged) {
    background: var(--ev2-bg);
    color: var(--ev2-fg);
  }

  .admin :global(.mono) {
    font-family: var(--font-mono);
    font-size: 0.92em;
  }

  .admin :global(.error) {
    color: var(--accent);
    font-weight: 600;
  }

  .admin :global(input[type='text']),
  .admin :global(input[type='date']),
  .admin :global(select),
  .admin :global(textarea) {
    font: inherit;
    font-size: 1rem;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 0.45rem 0.6rem;
    min-height: 40px;
  }

  .admin :global(label) {
    font-weight: 600;
  }
</style>
