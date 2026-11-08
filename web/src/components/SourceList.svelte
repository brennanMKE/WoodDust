<script lang="ts">
  import type { Source } from '../lib/types';
  import { sourceHref } from '../lib/footnotes';

  let { sources, numbers }: { sources: Source[]; numbers: Map<string, number> } = $props();
</script>

<ol class="sources">
  {#each sources as s (s.key)}
    {@const href = sourceHref(s)}
    <li id="src-{s.key}" value={numbers.get(s.key)}>
      <span class="cite">{s.citation}</span>
      {#if href}
        <a class="link" {href} rel="external noopener" target="_blank">
          {s.doi ? `doi:${s.doi}` : s.pmid ? `PubMed ${s.pmid}` : 'Source'}<span class="visually-hidden"> (opens in a new tab)</span>
        </a>
      {/if}
    </li>
  {/each}
</ol>

<style>
  .sources {
    padding-left: 1.6rem;
    font-size: 0.92rem;
  }

  li {
    margin: 0.4rem 0;
    scroll-margin-top: 1rem;
  }

  li:target {
    background: var(--blue-soft);
    border-radius: 4px;
  }

  .link {
    margin-left: 0.35rem;
    word-break: break-word;
  }

  @media print {
    .sources {
      font-size: 6.5pt;
      line-height: 1.2;
      columns: 3;
      column-gap: 1.5em;
    }

    li {
      margin: 0.1rem 0;
      break-inside: avoid;
    }

    .link {
      display: none;
    }
  }
</style>
