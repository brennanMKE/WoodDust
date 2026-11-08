<script lang="ts">
  import { app } from '../lib/nav.svelte';
  import SearchBox from './SearchBox.svelte';

  let emblemFailed = $state(false);

  const links = [
    { href: '/browse', label: 'Browse woods' },
    { href: '/risks', label: 'Risks' },
    { href: '/protection', label: 'Protection' },
    { href: '/sources', label: 'Sources' },
  ];

  function current(href: string): boolean {
    const p = app.match.path;
    return p === href || p.startsWith(href + '/');
  }
</script>

<header class="site-header no-print">
  <div class="double-rule" aria-hidden="true"></div>
  <div class="wrap bar">
    <a class="brand" href="/" aria-label="{app.site.name} home">
      <!-- Art slot: images/art/emblem.png (site-art.md §2); tree-ring mark until it exists. -->
      {#if !emblemFailed}
        <img
          src="/images/art/emblem.png"
          alt="{app.site.name} emblem: Babe the Blue Ox's head above an open book."
          width="40"
          height="40"
          onerror={() => (emblemFailed = true)}
        />
      {:else}
        <svg viewBox="0 0 32 32" width="40" height="40" aria-hidden="true" focusable="false">
          <circle cx="16" cy="16" r="15" fill="var(--babe)" />
          <g fill="none" stroke="var(--on-babe)" stroke-width="1.8"><circle cx="16" cy="16" r="3" /><circle cx="16" cy="16" r="7" /><circle cx="16" cy="16" r="11" /></g>
        </svg>
      {/if}
      <span>{app.site.name}</span>
    </a>
    <nav aria-label="Main">
      <ul>
        {#each links as l (l.href)}
          <li><a href={l.href} aria-current={current(l.href) ? 'page' : undefined}>{l.label}</a></li>
        {/each}
      </ul>
    </nav>
    {#if app.match.id !== 'home'}
      <div class="header-search">
        <SearchBox variant="compact" id="header-search" />
      </div>
    {/if}
  </div>
</header>

<style>
  .site-header {
    background: var(--surface);
    border-bottom: 1px solid var(--rule);
  }

  .bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1.5rem;
    padding-top: 0.6rem;
    padding-bottom: 0.6rem;
  }

  .brand {
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    font-family: var(--font-serif);
    font-weight: 700;
    font-size: 1.4rem;
    letter-spacing: 0.02em;
    color: var(--text);
    text-decoration: none;
    min-height: 44px;
  }

  nav ul {
    display: flex;
    flex-wrap: wrap;
    gap: 0.1rem 0.25rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  nav a {
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    padding: 0 0.6rem;
    color: var(--text);
    text-decoration: none;
    font-weight: 550;
    border-radius: var(--radius-sm);
  }

  nav a:hover {
    background: var(--surface-2);
  }

  nav a[aria-current='page'] {
    text-decoration: underline;
    text-decoration-color: var(--accent);
    text-decoration-thickness: 3px;
    text-underline-offset: 0.4em;
  }

  .header-search {
    margin-left: auto;
    width: min(100%, 18rem);
  }

  @media (max-width: 48rem) {
    .header-search {
      width: 100%;
      margin-left: 0;
    }
  }
</style>
