<script lang="ts">
  import { asset } from '../lib/asset';
  // Printable shop poster (/print/shop-poster, contract §2). Content comes from
  // the page API like any other page; the layout is built to print on one
  // Letter or A4 sheet.
  import { app } from '../lib/nav.svelte';
  import type { ContentPage } from '../lib/types';
  import { keysFromHtml, numberKeys, orderedSources, sourceHref } from '../lib/footnotes';
  import { formatDate } from '../lib/format';
  import Art from '../components/Art.svelte';

  let { page }: { page: ContentPage } = $props();

  const numbers = $derived(numberKeys(keysFromHtml(page.html ?? '')));
  const sources = $derived(orderedSources(numbers, page.sources));
  // The Markdown may carry its own <h1> (the poster headline); then the page title is the kicker.
  const contentHasH1 = $derived(/<h1[\s>]/i.test(page.html ?? ''));
  const siteBase = $derived((app.site.base_url || location.origin).replace(/\/$/, ''));
</script>

<article class="wrap poster" aria-labelledby={contentHasH1 ? undefined : 'poster-title'}>
  <p class="actions no-print">
    <button class="button primary" type="button" onclick={() => window.print()}>Print this poster</button>
    <a class="button" href="/protection">Back to the protection guide</a>
  </p>

  <div class="sheet">
    <header class="poster-head">
      <!-- Art slot: images/art/poster-header.png (images/prompts/site-art.md §8). -->
      <Art
        src={asset('/images/art/poster-header.png')}
        alt="Paul Bunyan in full shop safety gear gives a thumbs-up beside Babe the Blue Ox."
        width={1536}
        height={1024}
        eager
        fallback="hide"
        class="poster-art"
      />
      <div class="double-rule" aria-hidden="true"></div>
      {#if contentHasH1}
        <p class="kicker">{app.site.name} · {page.title}</p>
      {:else}
        <p class="kicker">{app.site.name}</p>
        <h1 id="poster-title">{page.title}</h1>
      {/if}
    </header>

    <div class="poster-body">{@html page.html}</div>

    <footer class="poster-foot">
      <p class="disclaimer">
        Information only, not medical advice. See a doctor about symptoms. Full guide and sources: {siteBase}/protection
        {#if page.last_reviewed}· Reviewed {formatDate(page.last_reviewed)}{/if}
      </p>
      {#if sources.length}
        <ol class="poster-sources" aria-label="Sources">
          {#each sources as s (s.key)}
            {@const href = sourceHref(s)}
            <li id="src-{s.key}" value={numbers.get(s.key)}>
              {s.title ?? s.citation}{#if href}<span class="no-print"> · <a {href} rel="external noopener" target="_blank">link<span class="visually-hidden"> (opens in a new tab)</span></a></span>{/if}
            </li>
          {/each}
        </ol>
      {/if}
    </footer>
  </div>
</article>

<style>
  .poster {
    padding-top: 1.25rem;
    max-width: 52rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .sheet {
    background: var(--surface);
    border: 1px solid var(--rule);
    padding: clamp(1rem, 4vw, 2.25rem);
    box-shadow: var(--shadow);
  }

  .poster-head {
    text-align: center;
  }

  .poster-head :global(.poster-art) {
    max-width: 22rem;
    margin: 0 auto 0.75rem;
  }

  .kicker {
    font-size: 0.85rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--accent);
    margin: 0.6rem 0 0.25rem;
  }

  .poster-body :global(h1) {
    font-size: clamp(1.8rem, 1.3rem + 2vw, 2.6rem);
    text-align: center;
    margin: 0 0 0.4rem;
  }

  .poster-body :global(h1 + p) {
    text-align: center;
    font-size: 1.1rem;
    max-width: 38rem;
    margin: 0 auto 1rem;
  }

  /* The Markdown is flat (h2, list, h2, list…), so use columns rather than a grid. */
  .poster-body {
    column-gap: 2rem;
  }

  @media (min-width: 44rem) {
    .poster-body {
      columns: 2;
    }
  }

  .poster-body :global(h1),
  .poster-body :global(h1 + p) {
    column-span: all;
  }

  .poster-body :global(h2) {
    break-after: avoid;
  }

  .poster-body :global(ul),
  .poster-body :global(h2 + p) {
    break-inside: avoid;
  }

  /* Each "## n. Heading" + its list stays together in one cell. */
  .poster-body :global(h2) {
    font-size: 1.2rem;
    margin: 0.9rem 0 0.3rem;
    padding-bottom: 0.15rem;
    border-bottom: 2px solid var(--accent);
  }

  .poster-body :global(ul) {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .poster-body :global(li) {
    margin: 0.3rem 0;
    padding-left: 1.5em;
    text-indent: -1.5em;
  }

  .poster-body :global(input[type='checkbox']) {
    appearance: none;
    display: inline-block;
    width: 1em;
    height: 1em;
    margin: 0 0.5em 0 0;
    vertical-align: baseline;
    text-indent: 0;
    border: 2px solid var(--text);
    border-radius: 2px;
    transform: translateY(0.15em);
  }

  .poster-body :global(p:last-child) {
    margin-top: 1rem;
    text-align: center;
    font-weight: 600;
  }

  .poster-foot {
    margin-top: 1.25rem;
    padding-top: 0.6rem;
    border-top: 1px solid var(--rule);
    font-size: 0.85rem;
    color: var(--muted);
  }

  .disclaimer {
    margin: 0 0 0.4rem;
  }

  .poster-sources {
    margin: 0;
    padding-left: 1.4rem;
    columns: 2;
    column-gap: 1.5rem;
    font-size: 0.8rem;
  }

  .poster-sources li {
    break-inside: avoid;
  }

  /* One sheet on Letter or A4: with 10 mm margins the common printable box is
     about 196 × 259 mm (Letter's height, A4's width), so size for that. */
  @media print {
    @page {
      size: auto;
      margin: 10mm;
    }

    /* The poster carries its own disclaimer, so drop the site footer, but only on this page. */
    :global(body:has(.poster) .site-footer) {
      display: none;
    }

    .poster {
      padding: 0;
      max-width: none;
    }

    .sheet {
      border: 3px double #000;
      box-shadow: none;
      padding: 8mm 9mm;
      background: none;
    }

    .poster-head :global(.poster-art) {
      max-width: none;
      width: auto;
      height: 52mm;
      margin-bottom: 2mm;
    }

    .poster-head :global(.poster-art img) {
      width: auto;
      height: 100%;
    }

    .kicker {
      font-size: 8pt;
      color: #000;
      margin: 2mm 0 1mm;
    }

    .poster-body {
      columns: 2;
      column-gap: 8mm;
      font-size: 11pt;
      line-height: 1.3;
    }

    .poster-body :global(h1) {
      font-size: 24pt;
    }

    .poster-body :global(h1 + p) {
      font-size: 11.5pt;
      margin-bottom: 2mm;
    }

    .poster-body :global(h2) {
      font-size: 13pt;
      margin: 3mm 0 1mm;
      border-bottom-color: #000;
    }

    .poster-body :global(li) {
      margin: 0.8mm 0;
    }

    .poster-body :global(input[type='checkbox']) {
      border-color: #000;
    }

    .poster-body :global(sup) {
      font-size: 6pt;
    }

    .poster-foot {
      margin-top: 3mm;
      padding-top: 1.5mm;
      font-size: 7pt;
      color: #000;
    }

    .poster-sources {
      columns: 3;
      font-size: 6pt;
      line-height: 1.2;
    }
  }
</style>
