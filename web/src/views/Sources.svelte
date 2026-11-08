<script lang="ts">
  import type { Source, SourcesResponse } from '../lib/types';
  import { TIER_NAMES } from '../lib/labels';
  import { sourceHref } from '../lib/footnotes';
  import Art from '../components/Art.svelte';

  let { data }: { data: SourcesResponse } = $props();

  const tiers = $derived.by(() => {
    const by = new Map<string, Source[]>();
    for (const [key, s] of Object.entries(data?.sources ?? {})) {
      const arr = by.get(s.tier) ?? [];
      arr.push({ ...s, key });
      by.set(s.tier, arr);
    }
    return ['A', 'B', 'C', 'D']
      .filter((t) => by.has(t))
      .map((t) => ({ tier: t, items: by.get(t)!.sort((a, b) => a.citation.localeCompare(b.citation)) }));
  });
  const total = $derived(Object.keys(data?.sources ?? {}).length);
</script>

<div class="wrap">
  <h1>Sources</h1>
  <!-- Art slot: images/pages/methodology.jpg, shared with /methodology (site-art.md §6). -->
  <Art
    src="/images/pages/methodology.jpg"
    alt="Paul Bunyan stacks books and journals like lumber, checking each one against a ledger."
    width={1200}
    height={630}
    fallback="hide"
    class="plate sources-art"
  />
  <p class="lede">
    All {total} sources cited on this site. Agency listings and primary studies back the hazard claims; technical
    references are used only for names and non-health facts. Read how we grade evidence in the
    <a href="/methodology">methodology</a>.
  </p>

  {#if tiers.length > 1}
    <nav aria-label="Source groups">
      <ul class="jump">
        {#each tiers as t (t.tier)}<li><a href="#tier-{t.tier}">{TIER_NAMES[t.tier]} ({t.items.length})</a></li>{/each}
      </ul>
    </nav>
  {/if}

  {#each tiers as t (t.tier)}
    <section aria-labelledby="tier-{t.tier}">
      <h2 id="tier-{t.tier}">{TIER_NAMES[t.tier]}</h2>
      <ul class="list">
        {#each t.items as s (s.key)}
          {@const href = sourceHref(s)}
          <li id="src-{s.key}">
            <p class="cite">{s.citation}</p>
            <p class="meta">
              <span class="key">{s.key}</span>
              {#if href}
                <a {href} rel="external noopener" target="_blank">
                  {s.doi ? `doi:${s.doi}` : s.pmid ? `PubMed ${s.pmid}` : 'Open source'}<span class="visually-hidden"> (opens in a new tab)</span>
                </a>
              {/if}
              {#if s.verified}<span>Link checked{s.accessed ? ` ${s.accessed}` : ''}</span>{:else}<span>Not yet verified</span>{/if}
            </p>
          </li>
        {/each}
      </ul>
    </section>
  {/each}
</div>

<style>
  .lede {
    max-width: 46rem;
  }

  .wrap :global(.sources-art) {
    max-width: 36rem;
    margin: 0.5rem 0 1rem;
  }

  .jump {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1.25rem;
    padding: 0;
    list-style: none;
  }

  .jump a {
    display: inline-block;
    padding: 0.4rem 0;
  }

  .list {
    list-style: none;
    padding: 0;
    margin: 0;
    max-width: 50rem;
  }

  .list li {
    padding: 0.75rem 0;
    border-top: 1px solid var(--rule);
    scroll-margin-top: 1rem;
  }

  .list li:target {
    background: var(--blue-soft);
  }

  .cite {
    margin: 0 0 0.25rem;
  }

  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem 1rem;
    font-size: 0.88rem;
    color: var(--muted);
    margin: 0;
  }

  .key {
    font-family: var(--font-mono);
  }

  .meta a {
    word-break: break-word;
  }
</style>
