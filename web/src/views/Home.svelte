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

<!-- Hero: title, lede and search in one column with the illustration beside
     them on wide screens, so the featured woods start above the fold. On
     phones the search comes first and the picture follows, smaller. -->
<section class="hero" aria-labelledby="home-title">
  <div class="wrap hero-grid">
    <div class="hero-copy">
      <h1 id="home-title">Is this wood dust bad for me?</h1>
      <p class="lede">Look up a wood to see its known health hazards, how strong the evidence is, and where it comes from.</p>
      <div class="search-wrap">
        <SearchBox variant="hero" id="home-search" {initialQuery} labelHidden />
      </div>
    </div>
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
  </div>
</section>

{#if featured.length}
  <section class="wrap featured" aria-labelledby="featured-title">
    <div class="featured-head">
      <h2 id="featured-title">Most looked-up woods</h2>
      <a class="more" href="/browse">Browse all woods →</a>
    </div>
    <ul class="tiles">
      {#each featured as s (s.slug)}
        <li><SpeciesTile species={s} /></li>
      {/each}
    </ul>
  </section>
{/if}

<section class="wrap" aria-labelledby="all-dust-title">
  <div class="band">
    <div class="band-intro">
      <h2 id="all-dust-title">All wood dust is hazardous</h2>
      <p>
        Whatever the species, breathing wood dust over years can cause asthma and is linked to nasal cancer. The fine
        dust you can't see matters most. The good news: a few habits cut your exposure a lot.
      </p>
    </div>
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
  /* ---- Hero ---- */

  .hero {
    padding: clamp(0.75rem, 2vw, 1.5rem) 0 clamp(1rem, 2.5vw, 1.75rem);
  }

  .hero-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 0.75rem 2.5rem;
    align-items: center;
  }

  .hero-copy {
    max-width: 42rem;
  }

  .hero h1 {
    margin-bottom: 0.4rem;
  }

  .lede {
    color: var(--muted);
    font-size: 1.1rem;
    max-width: 34rem;
    margin: 0 0 1.1rem;
  }

  .search-wrap {
    max-width: 34rem;
  }

  /* Phone: the picture follows the search, small, so it never pushes the
     search or the featured woods down. */
  .hero :global(.hero-art) {
    width: min(100%, 15rem);
    justify-self: center;
    margin-top: 0.25rem;
  }

  @media (min-width: 48rem) {
    .hero-grid {
      grid-template-columns: minmax(0, 1fr) clamp(13rem, 30vw, 24rem);
    }

    .hero :global(.hero-art) {
      width: 100%;
      margin-top: 0;
    }
  }

  /* ---- Featured woods ---- */

  .featured-head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.25rem 1rem;
    margin-bottom: 0.75rem;
  }

  .featured-head h2 {
    margin: 0;
  }

  .more {
    font-weight: 600;
    white-space: nowrap;
    /* 44px tap target without changing the inline look. */
    padding: 0.55rem 0;
    margin: -0.55rem 0;
  }

  .tiles {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(10.5rem, 1fr));
    gap: 1rem;
  }

  .tiles li {
    display: flex;
  }

  .tiles li > :global(*) {
    flex: 1;
  }

  /* Five featured woods in one row on a laptop. */
  @media (min-width: 64rem) {
    .tiles {
      grid-template-columns: repeat(5, minmax(0, 1fr));
    }
  }

  /* ---- "All wood dust is hazardous" band ---- */

  .band {
    margin-top: 2rem;
    padding: 1.25rem clamp(1rem, 3vw, 1.75rem) 1.5rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-left: 6px solid var(--wood);
    border-radius: var(--radius);
  }

  .band h2 {
    margin-top: 0;
    margin-bottom: 0.4rem;
  }

  .band-intro p {
    max-width: 50rem;
    margin-bottom: 1rem;
  }

  .steps {
    list-style: none;
    padding: 0;
    margin: 0 0 1.25rem;
    display: grid;
    gap: 0.75rem 1.5rem;
  }

  .steps li {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.75rem;
    align-items: center;
  }

  .steps li > :global(.art) {
    width: 56px;
  }

  .steps li:not(:has(:global(.art))) {
    grid-template-columns: 1fr;
  }

  @media (min-width: 52rem) {
    .steps {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }

    .steps li {
      align-items: start;
    }

    .steps li > :global(.art) {
      margin-top: -0.2rem;
    }
  }

  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin: 0;
  }

  @media print {
    .hero :global(.hero-art),
    .steps li > :global(.art),
    .more {
      display: none;
    }

    .steps li {
      grid-template-columns: 1fr;
    }

    .band {
      border-left-width: 1px;
    }
  }
</style>
