<script lang="ts">
  // /posters/{id}/print: one poster, full page, for the browser's print
  // dialog ("Save as PDF"). A plaque built on /posters comes through in
  // memory (lib/posterDraft); a direct visit prints the stock poster.
  import { onMount } from 'svelte';
  import { drafts } from '../lib/posterDraft.svelte';
  import type { PosterData } from '../lib/types';
  import PosterSheet from '../components/PosterSheet.svelte';

  let { data }: { data: PosterData } = $props();
  const poster = $derived(data?.poster);
  const draft = $derived(poster ? drafts[poster.id] : undefined);
  const link = $derived(draft?.link ?? poster?.stock);

  onMount(() => {
    if (!poster || !drafts[poster.id]?.autoPrint) return;
    drafts[poster.id]!.autoPrint = false;
    // Let the art load before the dialog snapshots the page.
    const art = document.querySelector<SVGImageElement>('.print-sheet image[data-slot="art"]');
    const go = () => setTimeout(() => window.print(), 150);
    if (!art) return go();
    const img = new Image();
    img.onload = img.onerror = go;
    img.src = art.getAttribute('href') ?? '';
  });
</script>

{#if poster && link}
  <div class="wrap print-page">
    <div class="no-print bar">
      <h1>Print: {poster.title}</h1>
      <p>
        Choose Letter or A4 in the print dialog, with margins set to none if you can. To keep a PDF, choose “Save as
        PDF”.
      </p>
      <p class="actions">
        <button type="button" class="button primary" onclick={() => window.print()}>Print or save as PDF</button>
        <a class="button" href="/posters">Back to posters</a>
      </p>
    </div>
    <div class="print-sheet">
      <PosterSheet {poster} {link} plaque={draft?.plaque ?? null} />
    </div>
  </div>
{/if}

<style>
  .print-page {
    padding-top: 1rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .print-sheet {
    max-width: 28rem;
    margin: 1rem auto 0;
  }

  @media print {
    @page {
      size: auto;
      margin: 0;
    }

    :global(body:has(.print-sheet) .site-footer),
    :global(body:has(.print-sheet) .site-header) {
      display: none;
    }

    :global(body:has(.print-sheet)) {
      background: #ffffff;
      print-color-adjust: exact;
      -webkit-print-color-adjust: exact;
    }

    .print-page {
      padding: 0;
      max-width: none;
    }

    /* Fill the page height; the 2:3 sheet sits centred with even side margins. */
    .print-sheet {
      max-width: none;
      width: auto;
      height: 100vh;
      aspect-ratio: 2 / 3;
      margin: 0 auto;
      break-inside: avoid;
    }

    .print-sheet :global(.sheet) {
      box-shadow: none;
    }
  }
</style>
