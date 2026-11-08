<script lang="ts">
  import { app } from '../lib/nav.svelte';
  import type { SpeciesDetail } from '../lib/types';
  import { ACGIH_CLASSES, EVIDENCE, hazardName } from '../lib/labels';
  import { groupHazards, numberKeys, orderedSources, speciesCitationOrder } from '../lib/footnotes';
  import { formatDate } from '../lib/format';
  import EvidenceBadge from '../components/EvidenceBadge.svelte';
  import SpeciesImage from '../components/SpeciesImage.svelte';
  import SourceList from '../components/SourceList.svelte';
  import Art from '../components/Art.svelte';

  let { species: s }: { species: SpeciesDetail } = $props();

  const groups = $derived(groupHazards(s.hazards ?? []));
  const numbers = $derived(numberKeys(speciesCitationOrder(s, groups)));
  const sourceList = $derived(orderedSources(numbers, s.sources));
  const otherNames = $derived(
    [...new Set(s.common_names.slice(1).filter((n) => n.toLowerCase() !== s.name.toLowerCase()))],
  );
  const reg = $derived(s.regulatory ?? {});
  const hasSkin = $derived(s.hazards.some((h) => /^(skin|eye)_/.test(h.type) || h.type === 'photosensitization'));
  const siteBase = $derived((app.site.base_url || location.origin).replace(/\/$/, ''));
  const cardUrl = $derived(`${siteBase}/wood/${s.slug}`);
  const generalCancer = $derived(s.hazards.find((h) => h.type === 'carcinogen' && h.evidence_level === 0));

  const fractionText = (f?: string) => (f ? `${f} fraction` : '');
  const fmt = (n: number) => `${n} mg/m³`;
</script>

{#snippet cites(keys: string[])}
  {#each [...new Set(keys)] as k (k)}
    {#if numbers.has(k)}<sup><a href="#src-{k}" aria-label="Source {numbers.get(k)}">{numbers.get(k)}</a></sup>{/if}
  {/each}
{/snippet}

<article class="wrap card" aria-labelledby="species-name">
  <nav class="crumbs no-print" aria-label="Breadcrumb">
    <ol>
      <li><a href="/">Home</a></li>
      <li><a href="/browse">Woods</a></li>
      <li><span aria-current="page">{s.name}</span></li>
    </ol>
  </nav>

  <p class="print-only print-head">{app.site.name} · shop card · {cardUrl}</p>

  <header class="card-head">
    <div class="titles">
      <p class="eyebrow">{s.wood_type === 'softwood' ? 'Softwood' : 'Hardwood'} · {s.family}</p>
      <h1 id="species-name">{s.name}</h1>
      <p class="botanical-line">
        <span class="botanical">{s.botanical_name}</span>
        {#if s.synonyms?.length}
          <span class="muted">(also <span class="botanical">{s.synonyms.join(', ')}</span>)</span>
        {/if}
      </p>
      {#if otherNames.length}
        <p class="other-names"><span class="muted">Also sold as</span> {otherNames.join(', ')}</p>
      {/if}
      <p class="region muted no-print">{s.region}</p>
      <p class="actions no-print">
        <button class="button" type="button" onclick={() => window.print()}>
          <svg viewBox="0 0 20 20" width="18" height="18" aria-hidden="true" focusable="false"><path d="M5 7V2.5h10V7M5 14H3V8.5A1.5 1.5 0 0 1 4.5 7h11A1.5 1.5 0 0 1 17 8.5V14h-2M5 11.5h10v6H5z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" /></svg>
          Print shop card
        </button>
        <a class="button" href="/submit/{s.slug}">Report a correction</a>
      </p>
    </div>
    <figure class="photo">
      <SpeciesImage image={s.image} eager sizes="(min-width: 60rem) 26rem, 100vw" />
    </figure>
  </header>

  {#if s.description}
    <p class="description no-print">{s.description}</p>
  {/if}

  <div class="notice absence">
    <p class="no-print">
      <strong>No wood is “safe” to breathe.</strong> All wood dust can cause irritation, asthma and cancer. A hazard
      missing from this card means we found no reports, not that the wood is safe. Absence of reports is not evidence
      of safety.
    </p>
    <p class="print-only">
      <strong>All wood dust: cancer risk.</strong> IARC classifies all wood dust as carcinogenic to humans (mainly nasal
      and sinus cancer){#if generalCancer}{@render cites(generalCancer.sources)}{/if}; it can also cause asthma and
      irritation. No wood is “safe”: absence of reports is not evidence of safety. More: {siteBase}/risks
    </p>
  </div>

  <section aria-labelledby="hazards-title">
    <h2 id="hazards-title">Known hazards</h2>
    <p class="key muted no-print">
      Evidence levels: {#each [1, 2, 3, 0] as l, i (l)}{i ? ' · ' : ''}<EvidenceBadge level={l} short />{/each}
      <a href="/methodology">What these mean</a>
    </p>
    <div class="hazard-cols">
    {#each groups as g (g.type)}
      <section class="hazard-group" class:all-general={g.hazards.every((h) => h.evidence_level === 0)} aria-labelledby="hz-{g.type}">
        <h3 id="hz-{g.type}">{hazardName(g.type)}</h3>
        <ul class="hazards">
          {#each g.hazards as h, i (i)}
            <li class:general={h.evidence_level === 0}>
              <p class="badge-line"><EvidenceBadge level={h.evidence_level} /></p>
              <p class="summary">{h.summary}{@render cites(h.sources)}</p>
              <p class="facts muted">
                {#if h.effects?.length}<span><span class="k">Effects:</span> {h.effects.join(', ')}</span>{/if}
                {#if h.route?.length}<span><span class="k">Exposure:</span> {h.route.join(', ')}</span>{/if}
              </p>
            </li>
          {/each}
        </ul>
      </section>
    {/each}
    </div>
  </section>

  <section aria-labelledby="limits-title" class="limits">
    <h2 id="limits-title">Exposure limits and classification</h2>
    {#if s.carcinogen_class?.length}
      <ul class="classes">
        {#each s.carcinogen_class as c, i (i)}
          <li>
            <strong>{c.body} {c.class}</strong>{#if c.body === 'ACGIH' && ACGIH_CLASSES[c.class]}: {ACGIH_CLASSES[c.class]}{/if}{@render cites([c.source])}
          </li>
        {/each}
      </ul>
    {/if}
    <table>
      <caption class="visually-hidden">Occupational exposure limits for {s.name} dust</caption>
      <thead>
        <tr><th scope="col">Limit</th><th scope="col">Value (time-weighted average)</th><th scope="col">Notes</th></tr>
      </thead>
      <tbody>
        {#each s.exposure_limits ?? [] as l, i (i)}
          <tr>
            <th scope="row">{l.body}</th>
            <td>{fmt(l.value_mg_m3)}{@render cites([l.source])}</td>
            <td>{[fractionText(l.fraction), l.note].filter(Boolean).join('; ')}</td>
          </tr>
        {/each}
        {#if reg.niosh_rel_mg_m3 != null}
          <tr><th scope="row">NIOSH REL (US)</th><td>{fmt(reg.niosh_rel_mg_m3)}</td><td>Recommended limit for all wood dust; up to a 10-hour workday</td></tr>
        {/if}
        {#if s.wood_type === 'hardwood' && reg.eu_hardwood_mg_m3 != null}
          <tr><th scope="row">EU binding limit</th><td>{fmt(reg.eu_hardwood_mg_m3)}</td><td>Hardwood dust, 8-hour; also covers other wood dust mixed with it</td></tr>
        {/if}
        {#if s.wood_type === 'hardwood' && reg.uk_hardwood_mg_m3 != null}
          <tr><th scope="row">UK WEL</th><td>{fmt(reg.uk_hardwood_mg_m3)}</td><td>Hardwood dust, 8-hour</td></tr>
        {/if}
        {#if s.wood_type === 'softwood' && reg.uk_softwood_mg_m3 != null}
          <tr><th scope="row">UK WEL</th><td>{fmt(reg.uk_softwood_mg_m3)}</td><td>Softwood dust, 8-hour; the hardwood limit applies if mixed with hardwood dust</td></tr>
        {/if}
      </tbody>
    </table>
    <p class="muted small">
      These are workplace limits, not safe levels. Keep dust as low as you reasonably can.
      <a href="/limits">How the limits compare</a>.
    </p>
  </section>

  {#if s.known_compounds?.length}
    <section aria-labelledby="compounds-title" class="no-print">
      <h2 id="compounds-title">Compounds linked to effects</h2>
      <ul>
        {#each s.known_compounds as c, i (i)}
          <li><strong>{c.name}</strong>{c.role ? `: ${c.role}` : ''}{@render cites([c.source])}</li>
        {/each}
      </ul>
    </section>
  {/if}

  {#if s.related_species_note}
    <p class="related no-print"><strong>Related woods:</strong> {s.related_species_note}</p>
  {/if}

  <section aria-labelledby="protect-title" class="protect">
    <div class="protect-head">
      <Art src="/images/art/spot-respirator.png" alt="A half-face respirator." width={56} height={56} fallback="hide" class="no-print" />
      <h2 id="protect-title">Protect yourself</h2>
    </div>
    <ul>
      <li><a href="/protection/dust-collection">Collect dust at the tool</a>, especially when sanding.</li>
      <li><a href="/protection/respirators">Wear a fitted respirator</a> (N95 or better) for sanding and dusty jobs.</li>
      {#if hasSkin}
        <li><a href="/protection/skin-and-eyes">Cover skin and eyes</a>; this wood is reported to affect them.</li>
      {/if}
      <li><a href="/protection/cleanup">Clean up with a HEPA vacuum</a>, never compressed air.</li>
      <li><a href="/symptoms">Know the symptoms</a> and see a doctor if they show up.</li>
    </ul>
    <p class="no-print"><a href="/protection">The full protection guide</a></p>
  </section>

  {#if sourceList.length}
    <section aria-labelledby="sources-title">
      <h2 id="sources-title">Sources</h2>
      <SourceList sources={sourceList} {numbers} />
    </section>
  {/if}

  <footer class="card-foot">
    <p>
      Last reviewed <time datetime={s.last_reviewed}>{formatDate(s.last_reviewed)}</time>.
      <span class="no-print">Spotted an error or a missing source? <a href="/submit/{s.slug}">Report a correction</a>.</span>
    </p>
    <p class="print-only small key-line">
      Evidence levels: {EVIDENCE[1].label}; {EVIDENCE[2].label}; {EVIDENCE[3].label}; General = applies to all wood dust.
    </p>
  </footer>
</article>

<style>
  .card {
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

  .card-head {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.25rem;
    align-items: start;
  }

  @media (min-width: 52rem) {
    .card-head {
      grid-template-columns: 1fr 26rem;
    }
  }

  .titles p {
    margin: 0 0 0.3rem;
  }

  .botanical-line {
    font-size: 1.15rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 1rem !important;
  }

  .photo {
    margin: 0;
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: var(--shadow);
  }

  .description {
    font-size: 1.08rem;
    max-width: 46rem;
    margin-top: 1.5rem;
  }

  .absence {
    max-width: 46rem;
  }

  .key {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.9rem;
  }

  .hazard-group {
    border-top: 1px solid var(--rule);
    max-width: 50rem;
  }

  .hazard-group h3 {
    margin-top: 1rem;
  }

  .hazards {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .hazards li {
    padding: 0.4rem 0 0.9rem;
  }

  .hazards li + li {
    border-top: 1px dashed var(--rule);
    padding-top: 0.8rem;
  }

  .badge-line {
    margin: 0 0 0.35rem;
  }

  .summary {
    margin: 0 0 0.25rem;
  }

  .facts {
    display: flex;
    flex-wrap: wrap;
    gap: 0.2rem 1.25rem;
    font-size: 0.9rem;
    margin: 0;
  }

  .k {
    font-weight: 600;
  }

  .classes {
    padding-left: 1.2rem;
  }

  table {
    border-collapse: collapse;
    width: 100%;
    max-width: 46rem;
    font-size: 0.95rem;
  }

  th,
  td {
    text-align: left;
    padding: 0.5rem 0.6rem;
    border-bottom: 1px solid var(--rule);
    vertical-align: top;
  }

  thead th {
    font-size: 0.85rem;
    color: var(--muted);
  }

  .small {
    font-size: 0.9rem;
  }

  .limits .small {
    margin-top: 0.6rem;
  }

  .related {
    max-width: 46rem;
    margin-top: 1.5rem;
  }

  .protect ul {
    padding-left: 1.2rem;
  }

  .protect-head {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 2rem;
  }

  .protect-head :global(.art) {
    width: 56px;
    flex: none;
  }

  .protect-head h2 {
    margin: 0;
  }

  .card-foot {
    margin-top: 2rem;
    padding-top: 1rem;
    border-top: 1px solid var(--rule);
    color: var(--muted);
  }

  .print-only {
    display: none;
  }

  /* One-page shop card. */
  @media print {
    @page {
      size: auto;
      margin: 10mm;
    }

    .card {
      padding: 0;
      max-width: none;
    }

    .print-only {
      display: block;
    }

    .description,
    .absence,
    .hazard-group,
    .related,
    table {
      max-width: none;
    }

    .print-head {
      font-size: 8pt;
      margin: 0 0 4pt;
      border-bottom: 1px solid #000;
      padding-bottom: 2pt;
    }

    .card-head {
      display: block;
    }

    .photo {
      display: none;
    }

    .protect-head {
      margin-top: 6pt;
    }

    h1 {
      font-size: 18pt;
      margin: 0;
    }

    h2 {
      font-size: 11pt;
      margin: 5pt 0 2pt;
    }

    h3 {
      font-size: 9.5pt;
      margin: 3pt 0 1pt;
    }

    .titles p,
    .description {
      margin: 0 0 2pt;
      font-size: 9pt;
    }

    .absence {
      margin: 5pt 0;
      padding: 4pt 6pt;
      font-size: 8.5pt;
      background: none;
      border: 1px solid #000;
      border-left-width: 3pt;
    }

    .hazard-group {
      break-inside: avoid;
      border-top: 0;
    }

    .hazards li,
    .hazards li + li {
      padding: 0 0 2pt;
      border: 0;
      font-size: 8pt;
    }

    .badge-line {
      display: inline;
      margin-right: 4pt;
    }

    .summary {
      display: inline;
    }

    .facts {
      display: none;
    }

    .hazard-cols {
      columns: 2;
      column-gap: 14pt;
    }

    .hazard-group h3:first-child {
      margin-top: 0;
    }

    /* General (all-wood-dust) hazards are the same on every card and are
       covered by the notice above, so the shop card leaves them out. */
    .hazards li.general,
    .hazard-group.all-general {
      display: none;
    }

    table {
      font-size: 7.5pt;
    }

    thead {
      display: none;
    }

    th,
    td {
      padding: 1.5pt 4pt;
    }

    .protect ul {
      margin: 0;
      padding: 0;
      list-style: none;
      font-size: 8pt;
    }

    .protect li {
      display: inline;
    }

    .protect li + li::before {
      content: ' • ';
    }

    .limits .small,
    .card-foot .print-only {
      display: none;
    }

    .card-foot {
      margin-top: 4pt;
      padding-top: 2pt;
      font-size: 8pt;
    }

    .card-foot p {
      margin: 0;
    }
  }
</style>
