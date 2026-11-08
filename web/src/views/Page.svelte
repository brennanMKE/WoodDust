<script lang="ts">
  import type { ContentPage } from '../lib/types';
  import { keysFromHtml, numberKeys, orderedSources } from '../lib/footnotes';
  import { formatDate } from '../lib/format';
  import SourceList from '../components/SourceList.svelte';
  import Art from '../components/Art.svelte';

  let { page }: { page: ContentPage } = $props();

  const numbers = $derived(numberKeys(keysFromHtml(page.html ?? '')));
  const sources = $derived(orderedSources(numbers, page.sources));
  const toc = $derived((page.toc ?? []).filter((t) => t.level <= 3));
  const isProtection = $derived(page.slug?.startsWith('protection'));
  const crumbs = $derived(
    page.slug?.startsWith('protection/') ? [{ href: '/protection', label: 'Protection' }] : [],
  );
</script>

<article class="wrap page" aria-labelledby="page-title">
  <nav class="crumbs" aria-label="Breadcrumb">
    <ol>
      <li><a href="/">Home</a></li>
      {#each crumbs as c (c.href)}<li><a href={c.href}>{c.label}</a></li>{/each}
      <li><span aria-current="page">{page.title}</span></li>
    </ol>
  </nav>

  <header>
    <h1 id="page-title">{page.title}</h1>
    {#if page.description}<p class="lede">{page.description}</p>{/if}
  </header>

  <div class="layout" class:has-toc={toc.length > 1}>
    {#if toc.length > 1}
      <nav class="toc" aria-labelledby="toc-title">
        <h2 id="toc-title">On this page</h2>
        <ol>
          {#each toc as t (t.id)}
            <li class="lvl{t.level}"><a href="#{t.id}">{t.text}</a></li>
          {/each}
        </ol>
      </nav>
    {/if}

    <div class="main-col">
      {#if page.image?.url && !page.image.url.endsWith('/placeholder.svg')}
        <!-- Art slot: images/pages/<page>.jpg (images/prompts/site-art.md §6). -->
        <Art src={page.image.url} alt={page.image.alt} width={1200} height={630} eager fallback="hide" class="hero-img" />
      {/if}
      <!-- Server-rendered from reviewed Markdown in content/pages (contract §3). -->
      <div class="prose">{@html page.html}</div>

      {#if isProtection && page.slug !== 'protection'}
        <p class="back"><a href="/protection">← All protection topics</a></p>
      {/if}

      {#if sources.length}
        <section aria-labelledby="page-sources">
          <h2 id="page-sources">Sources</h2>
          <SourceList {sources} {numbers} />
        </section>
      {/if}

      {#if page.last_reviewed}
        <p class="muted reviewed">Last reviewed <time datetime={page.last_reviewed}>{formatDate(page.last_reviewed)}</time>.</p>
      {/if}
    </div>
  </div>
</article>

<style>
  .page {
    padding-top: 1rem;
  }

  .crumbs ol {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    list-style: none;
    margin: 0 0 1rem;
    padding: 0;
    font-size: 0.92rem;
  }

  .crumbs li + li::before {
    content: '›';
    margin-right: 0.35rem;
    color: var(--muted);
  }

  .lede {
    font-size: 1.15rem;
    color: var(--muted);
    max-width: 44rem;
  }

  .layout.has-toc {
    display: grid;
    gap: 2rem;
  }

  @media (min-width: 60rem) {
    .layout.has-toc {
      grid-template-columns: 15rem 1fr;
    }

    .toc {
      position: sticky;
      top: 1rem;
      align-self: start;
    }
  }

  .toc {
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
    padding: 0.75rem 1rem;
    font-size: 0.92rem;
  }

  .toc h2 {
    font-size: 1rem;
    margin: 0 0 0.4rem;
  }

  .toc ol {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .toc li {
    margin: 0.2rem 0;
  }

  .toc .lvl3 {
    padding-left: 0.9rem;
  }

  .main-col :global(.hero-img) {
    margin: 0 0 1.5rem;
    max-width: 44rem;
    border-radius: var(--radius);
    overflow: hidden;
  }

  .reviewed {
    margin-top: 2rem;
    font-size: 0.92rem;
  }

  @media print {
    .crumbs,
    .toc,
    .main-col :global(.hero-img),
    .back {
      display: none;
    }

    .layout.has-toc {
      display: block;
    }
  }
</style>
