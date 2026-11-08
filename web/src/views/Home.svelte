<script lang="ts">
  import { asset } from '../lib/asset';
  import { app } from '../lib/nav.svelte';
  import type { SpeciesList } from '../lib/types';
  import SearchBox from '../components/SearchBox.svelte';
  import SpeciesTile from '../components/SpeciesTile.svelte';
  import Art from '../components/Art.svelte';

  let { data }: { data: SpeciesList } = $props();
  const featured = $derived(
    (data?.species ?? [])
      .filter((s) => s.featured_rank != null)
      .sort((a, b) => (a.featured_rank ?? 99) - (b.featured_rank ?? 99))
      .slice(0, 5),
  );
  const initialQuery = new URLSearchParams(app.search).get('q') ?? '';
</script>

<section class="hero" aria-labelledby="home-title">
  <div class="wrap hero-inner">
    <!-- Art slot: images/mascots/hero.png (images/prompts/site-art.md §1). -->
    <Art
      src={asset('/images/mascots/hero.png')}
      alt="Paul Bunyan, wearing a respirator and safety glasses, sands a board at a workbench while Babe the Blue Ox watches a dust collector hose pull the dust away."
      width={1536}
      height={1024}
      eager
      fallback="hide"
      class="hero-art"
    />
    <h1 id="home-title">Is this wood dust bad for me?</h1>
    <p class="lede">Look up a wood to see its known health hazards, how strong the evidence is, and where it comes from.</p>
    <div class="search-wrap">
      <SearchBox variant="hero" id="home-search" {initialQuery} />
    </div>
  </div>
</section>

{#if featured.length}
  <section class="wrap featured" aria-labelledby="featured-title">
    <h2 id="featured-title">Most looked-up woods</h2>
    <ul class="tiles">
      {#each featured as s (s.slug)}
        <li><SpeciesTile species={s} /></li>
      {/each}
    </ul>
    <p class="more"><a href="/browse">Browse all woods →</a></p>
  </section>
{/if}

<section class="wrap" aria-labelledby="all-dust-title">
  <div class="band">
    <h2 id="all-dust-title">All wood dust is hazardous</h2>
    <p>
      Whatever the species, breathing wood dust over years can cause asthma and is linked to nasal cancer. The fine
      dust you can't see matters most. The good news: a few habits cut your exposure a lot.
    </p>
    <ul class="steps">
      <li>
        <Art src={asset('/images/art/spot-dust-collector.png')} alt="A barrel dust collector." width={72} height={72} fallback="hide" />
        <span><strong>Catch it at the tool.</strong> Dust collection or extraction where the dust is made.</span>
      </li>
      <li>
        <Art src={asset('/images/art/spot-respirator.png')} alt="A half-face respirator." width={72} height={72} fallback="hide" />
        <span><strong>Wear a fitted respirator</strong> for sanding and dusty jobs.</span>
      </li>
      <li>
        <Art src={asset('/images/art/spot-babe-hoofprint.png')} alt="Babe's hoofprint." width={72} height={72} fallback="hide" />
        <span><strong>Clean up with a HEPA vacuum,</strong> never compressed air or a dry broom.</span>
      </li>
    </ul>
    <p class="links">
      <a class="button" href="/risks">Why wood dust is a risk</a>
      <a class="button primary" href="/protection">How to protect yourself</a>
    </p>
  </div>
</section>

<style>
  .hero {
    padding: clamp(0.75rem, 2.5vw, 1.75rem) 0 2rem;
    text-align: center;
  }

  .hero-inner {
    max-width: 46rem;
  }

  /* Small enough that the search box stays above the fold: at most 400px
     wide and never taller than about a quarter of the window (3:2 art). */
  .hero :global(.hero-art) {
    width: min(100%, 400px, 40vh);
    margin: 0 auto 0.25rem;
  }

  .lede {
    color: var(--muted);
    font-size: 1.12rem;
    max-width: 36rem;
    margin: 0 auto 1.5rem;
  }

  .search-wrap {
    max-width: 38rem;
    margin: 0 auto;
  }

  .featured h2 {
    text-align: center;
  }

  .tiles {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(11.5rem, 1fr));
    gap: 1rem;
  }

  .tiles li {
    display: flex;
  }

  .tiles li > :global(*) {
    flex: 1;
  }

  .more {
    text-align: center;
    margin-top: 1rem;
  }

  .band {
    margin-top: 2.5rem;
    padding: 1.5rem clamp(1rem, 4vw, 2rem);
    background: var(--surface);
    border: 1px solid var(--rule);
    border-left: 6px solid var(--wood);
    border-radius: var(--radius);
  }

  .band h2 {
    margin-top: 0;
  }

  .steps {
    list-style: none;
    padding: 0;
    margin: 0 0 1.25rem;
  }

  .steps li {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.75rem;
    align-items: center;
    gap: 1rem;
    margin: 0.75rem 0;
    min-height: 72px;
  }

  .steps li > :global(.art) {
    width: 72px;
  }

  .steps li:not(:has(:global(.art))) {
    grid-template-columns: 1fr;
  }

  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin: 0;
  }
</style>
