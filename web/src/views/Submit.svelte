<script lang="ts">
  import { app } from '../lib/nav.svelte';

  let { slug }: { slug?: string } = $props();
  const name = $derived(new URLSearchParams(app.search).get('name') ?? '');
</script>

<div class="wrap narrow">
  <h1>{slug ? 'Report a correction' : 'Suggest a wood'}</h1>
  <div class="notice">
    <p>
      <strong>Coming soon.</strong> Signed-in contributors will be able to
      {slug ? 'send corrections and better sources for this card' : `suggest${name ? ` “${name}”` : ' a wood we have not listed'}`}
      here. Every change is reviewed and must cite a source with a DOI, PubMed ID or an official agency page.
    </p>
  </div>
  <p>
    {#if slug}<a href="/wood/{slug}">← Back to the card</a>{:else}<a href="/">← Back to search</a>{/if}
  </p>
  <p class="muted">
    If you are reacting to a wood, this site can't help with that: please see a doctor and tell them which woods you
    work with.
  </p>
</div>

<style>
  .narrow {
    max-width: 44rem;
    padding-top: 1.5rem;
  }
</style>
