<script lang="ts">
  // One shop poster, drawn from its SVG template with the slots filled
  // (lib/poster.ts). Presentation attributes only, so it is safe to inline
  // under the strict CSP. Missing art leaves the template's placeholder block.
  import { fillTemplateString, posterLabel } from '../lib/poster';
  import type { PosterLink, PosterView } from '../lib/types';

  let { poster, link, plaque = null }: { poster: PosterView; link: PosterLink; plaque?: string | null } = $props();

  const art = $derived(poster.art.available ? poster.art.url : null);
  const markup = $derived(
    fillTemplateString(poster.template, { art, qrSVG: link.qr_svg, plaque }, posterLabel(poster, plaque)),
  );
</script>

<div class="sheet">
  {#if markup}
    {@html markup}
  {:else}
    <p class="muted">This poster can’t be shown here.</p>
  {/if}
</div>

<style>
  .sheet {
    aspect-ratio: 2 / 3;
    width: 100%;
    background: var(--surface-2);
    box-shadow: var(--shadow);
    line-height: 0;
  }

  .sheet :global(svg) {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
