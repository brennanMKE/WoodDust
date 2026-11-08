<script lang="ts">
  import type { SpeciesSummary } from '../lib/types';
  import { HAZARD_TYPES, hazardName } from '../lib/labels';
  import EvidenceBadge from './EvidenceBadge.svelte';
  import SpeciesImage from './SpeciesImage.svelte';

  let { species, headingLevel = 3 }: { species: SpeciesSummary; headingLevel?: 2 | 3 } = $props();
  const types = $derived(HAZARD_TYPES.filter((t) => species.hazard_types.includes(t)));
</script>

<article class="tile">
  <SpeciesImage image={species.image} sizes="(min-width: 60rem) 14rem, (min-width: 40rem) 45vw, 100vw" />
  <div class="body">
    <svelte:element this={headingLevel === 2 ? 'h2' : 'h3'} class="name">
      <a href={species.url}>{species.name}</a>
    </svelte:element>
    <p class="botanical">{species.botanical_name}</p>
    <p class="strongest">
      <span class="visually-hidden">Strongest evidence:</span>
      <EvidenceBadge level={species.max_evidence_level ?? 0} short />
    </p>
    {#if types.length}
      <p class="hazards">{types.slice(0, 3).map(hazardName).join(' · ')}{types.length > 3 ? ` +${types.length - 3} more` : ''}</p>
    {/if}
  </div>
</article>

<style>
  .tile {
    position: relative;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: var(--shadow);
    display: flex;
    flex-direction: column;
    transition: transform 0.15s ease;
  }

  .tile:hover {
    transform: translateY(-2px);
  }

  .tile:focus-within {
    outline: 3px solid var(--focus);
    outline-offset: 2px;
  }

  .body {
    padding: 0.75rem 1rem 1rem;
  }

  .name {
    font-size: 1.2rem;
    margin: 0;
  }

  .name a {
    color: var(--text);
    text-decoration: none;
  }

  /* Whole tile is clickable; the link stays the single tab stop. */
  .name a::after {
    content: '';
    position: absolute;
    inset: 0;
  }

  .name a:focus-visible {
    outline: none;
  }

  p {
    margin: 0.15rem 0 0;
  }

  .botanical {
    color: var(--muted);
    font-size: 0.92rem;
  }

  .strongest {
    margin-top: 0.5rem;
  }

  .hazards {
    margin-top: 0.4rem;
    font-size: 0.88rem;
    color: var(--muted);
  }
</style>
