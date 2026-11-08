<script lang="ts">
  import { app, replaceSearch } from '../lib/nav.svelte';
  import type { SpeciesSummary } from '../lib/types';
  import { EVIDENCE, evidenceRank, HAZARD_TYPES, hazardName } from '../lib/labels';
  import EvidenceBadge from '../components/EvidenceBadge.svelte';
  import Art from '../components/Art.svelte';

  let { data }: { data: { species: SpeciesSummary[] } } = $props();

  type SortKey = 'name' | 'botanical_name' | 'wood_type' | 'evidence' | 'hazards';
  const SORTS: Record<SortKey, string> = {
    name: 'Name',
    botanical_name: 'Botanical name',
    wood_type: 'Wood type',
    evidence: 'Strongest evidence',
    hazards: 'Number of hazard types',
  };

  const params = new URLSearchParams(app.search);
  let hazard = $state(params.get('hazard_type') ?? '');
  let evidence = $state(params.get('evidence_level') ?? '');
  let woodType = $state(params.get('wood_type') ?? '');
  let text = $state(params.get('q') ?? '');
  let sortKey = $state<SortKey>((params.get('sort') as SortKey) in SORTS ? (params.get('sort') as SortKey) : 'name');
  let sortDir = $state<'asc' | 'desc'>(params.get('dir') === 'desc' ? 'desc' : 'asc');

  const norm = (s: string) =>
    s
      .normalize('NFD')
      .replace(/[̀-ͯ]/g, '')
      .toLowerCase();

  const all = $derived(data?.species ?? []);

  const rows = $derived.by(() => {
    const t = norm(text.trim());
    const out = all.filter((s) => {
      if (hazard && !s.hazard_types.includes(hazard as never)) return false;
      if (woodType && s.wood_type !== woodType) return false;
      if (evidence === 'general' && s.max_evidence_level !== null) return false;
      if (evidence && evidence !== 'general' && String(s.max_evidence_level) !== evidence) return false;
      if (t && !norm([s.name, s.botanical_name, ...s.common_names, s.region].join(' ')).includes(t)) return false;
      return true;
    });
    const dir = sortDir === 'asc' ? 1 : -1;
    const cmp = (a: SpeciesSummary, b: SpeciesSummary): number => {
      switch (sortKey) {
        case 'botanical_name':
          return a.botanical_name.localeCompare(b.botanical_name);
        case 'wood_type':
          return a.wood_type.localeCompare(b.wood_type);
        case 'evidence':
          return evidenceRank(a.max_evidence_level) - evidenceRank(b.max_evidence_level);
        case 'hazards':
          return b.hazard_types.length - a.hazard_types.length;
        default:
          return a.name.localeCompare(b.name);
      }
    };
    return out.sort((a, b) => cmp(a, b) * dir || a.name.localeCompare(b.name));
  });

  // Reflect filters in the URL (without adding history entries) so a view can be shared.
  $effect(() => {
    const p = new URLSearchParams();
    if (hazard) p.set('hazard_type', hazard);
    if (evidence) p.set('evidence_level', evidence);
    if (woodType) p.set('wood_type', woodType);
    if (text.trim()) p.set('q', text.trim());
    if (sortKey !== 'name') p.set('sort', sortKey);
    if (sortDir !== 'asc') p.set('dir', sortDir);
    const s = p.toString();
    replaceSearch(s ? `?${s}` : '');
  });

  function sortBy(k: SortKey) {
    if (sortKey === k) sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    else {
      sortKey = k;
      sortDir = 'asc';
    }
  }

  function ariaSort(k: SortKey): 'ascending' | 'descending' | 'none' {
    return sortKey === k ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none';
  }

  function reset() {
    hazard = evidence = woodType = text = '';
    sortKey = 'name';
    sortDir = 'asc';
  }

  const typesFor = (s: SpeciesSummary) => HAZARD_TYPES.filter((t) => s.hazard_types.includes(t));
</script>

<div class="wrap">
  <h1>Browse woods</h1>
  <!-- Art: the site's default share image (Paul and Babe reading the almanac). -->
  <Art
    src="/images/og-default.jpg"
    alt="Paul Bunyan and Babe the Blue Ox reading a giant almanac in a woodshop."
    width={1200}
    height={630}
    fallback="hide"
    class="plate browse-art"
  />
  <p class="muted lede">
    {all.length} woods. Every one also carries the hazards shared by all wood dust. Filter by what you need to know.
  </p>

  <form class="filters" aria-label="Filter woods" onsubmit={(e) => e.preventDefault()}>
    <div class="f">
      <label for="f-text">Name contains</label>
      <input id="f-text" type="search" bind:value={text} autocomplete="off" />
    </div>
    <div class="f">
      <label for="f-hazard">Hazard</label>
      <select id="f-hazard" bind:value={hazard}>
        <option value="">Any hazard</option>
        {#each HAZARD_TYPES as t (t)}
          <option value={t}>{hazardName(t)}</option>
        {/each}
      </select>
    </div>
    <div class="f">
      <label for="f-evidence">Strongest evidence</label>
      <select id="f-evidence" bind:value={evidence}>
        <option value="">Any level</option>
        <option value="1">{EVIDENCE[1].label}</option>
        <option value="2">{EVIDENCE[2].label}</option>
        <option value="3">{EVIDENCE[3].label}</option>
        <option value="general">General hazards only</option>
      </select>
    </div>
    <div class="f">
      <label for="f-wood">Wood type</label>
      <select id="f-wood" bind:value={woodType}>
        <option value="">Hardwood and softwood</option>
        <option value="hardwood">Hardwood</option>
        <option value="softwood">Softwood</option>
      </select>
    </div>
    <div class="f sort-mobile">
      <label for="f-sort">Sort by</label>
      <select id="f-sort" bind:value={sortKey}>
        {#each Object.entries(SORTS) as [k, label] (k)}
          <option value={k}>{label}</option>
        {/each}
      </select>
    </div>
    <div class="f reset">
      <button class="button" type="button" onclick={reset}>Clear filters</button>
    </div>
  </form>

  <p class="count" role="status" aria-live="polite">
    Showing {rows.length} of {all.length} woods.
  </p>

  {#if rows.length}
    <table class="species-table">
      <caption class="visually-hidden">Woods and their hazards. Column headers with buttons can be sorted.</caption>
      <thead>
        <tr>
          {#each ['name', 'botanical_name', 'wood_type'] as const as k (k)}
            <th scope="col" aria-sort={ariaSort(k)}>
              <button type="button" onclick={() => sortBy(k)}>{SORTS[k]}<span class="arrow" aria-hidden="true">{sortKey === k ? (sortDir === 'asc' ? '▲' : '▼') : ''}</span></button>
            </th>
          {/each}
          <th scope="col" aria-sort={ariaSort('hazards')}>
            <button type="button" onclick={() => sortBy('hazards')}>Hazards<span class="arrow" aria-hidden="true">{sortKey === 'hazards' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</span></button>
          </th>
          <th scope="col" aria-sort={ariaSort('evidence')}>
            <button type="button" onclick={() => sortBy('evidence')}>{SORTS.evidence}<span class="arrow" aria-hidden="true">{sortKey === 'evidence' ? (sortDir === 'asc' ? '▲' : '▼') : ''}</span></button>
          </th>
        </tr>
      </thead>
      <tbody>
        {#each rows as s (s.slug)}
          <tr>
            <th scope="row" class="c-name"><a href={s.url}>{s.name}</a></th>
            <td class="c-bot"><span class="lbl">Botanical name: </span><span class="botanical">{s.botanical_name}</span></td>
            <td class="c-type"><span class="lbl">Type: </span>{s.wood_type === 'softwood' ? 'Softwood' : 'Hardwood'}</td>
            <td class="c-haz">
              <span class="lbl">Hazards: </span>
              {typesFor(s).map(hazardName).join(', ')}
            </td>
            <td class="c-ev"><span class="lbl">Strongest evidence: </span><EvidenceBadge level={s.max_evidence_level ?? 0} short /></td>
          </tr>
        {/each}
      </tbody>
    </table>
  {:else}
    <div class="notice">
      <p>No woods match those filters. <button class="linkish" type="button" onclick={reset}>Clear filters</button></p>
    </div>
  {/if}
</div>

<style>
  .wrap :global(.browse-art) {
    max-width: 36rem;
    margin: 0.5rem 0 1.25rem;
  }

  .lede {
    max-width: 44rem;
  }

  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem 1rem;
    align-items: end;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
    margin: 1rem 0;
  }

  .f {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 0;
    flex: 1 1 10rem;
  }

  .f.reset {
    flex: 0 0 auto;
    min-width: 0;
  }

  label {
    font-size: 0.88rem;
    font-weight: 600;
  }

  input,
  select {
    font: inherit;
    min-height: 44px;
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--bg);
    color: var(--text);
  }

  .count {
    font-size: 0.92rem;
    color: var(--muted);
  }

  .species-table {
    width: 100%;
    border-collapse: collapse;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  thead th {
    position: sticky;
    top: 0;
    background: var(--surface-2);
    text-align: left;
    font-size: 0.88rem;
  }

  thead button {
    font: inherit;
    font-weight: 700;
    color: var(--text);
    background: none;
    border: 0;
    padding: 0.6rem 0.75rem;
    min-height: 44px;
    width: 100%;
    text-align: left;
    cursor: pointer;
  }

  .arrow {
    display: inline-block;
    width: 1.2em;
    font-size: 0.75em;
    margin-left: 0.25em;
  }

  tbody th,
  tbody td {
    padding: 0.55rem 0.75rem;
    border-top: 1px solid var(--rule);
    text-align: left;
    vertical-align: top;
  }

  tbody tr:hover {
    background: var(--surface-2);
  }

  .c-name a {
    font-weight: 650;
  }

  .c-haz {
    font-size: 0.92rem;
    max-width: 22rem;
  }

  .lbl {
    display: none;
  }

  .sort-mobile {
    display: none;
  }

  .linkish {
    font: inherit;
    color: var(--blue);
    background: none;
    border: 0;
    padding: 0;
    text-decoration: underline;
    cursor: pointer;
  }

  /* Cards below the breakpoint: same table, re-laid out. */
  @media (max-width: 46rem) {
    .sort-mobile {
      display: flex;
    }

    .species-table,
    .species-table tbody,
    .species-table tr,
    .species-table th,
    .species-table td {
      display: block;
    }

    .species-table {
      background: none;
      border: 0;
    }

    .species-table thead {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
    }

    .species-table tr {
      background: var(--surface);
      border: 1px solid var(--rule);
      border-radius: var(--radius);
      padding: 0.75rem 1rem;
      margin-bottom: 0.75rem;
    }

    .species-table tbody th,
    .species-table tbody td {
      border: 0;
      padding: 0.15rem 0;
    }

    .c-name {
      font-size: 1.15rem;
    }

    .filters {
      display: grid;
      grid-template-columns: 1fr;
    }

    .lbl {
      display: inline;
      margin-right: 0.3em;
      color: var(--muted);
      font-size: 0.88rem;
    }

    .c-bot .lbl {
      display: none;
    }
  }
</style>
