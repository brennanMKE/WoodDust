<script lang="ts">
  // /posters (contract §2): the shop posters, their print files and the
  // "Posted by" plaque builder (docs/posters.md).
  import { freshFormToken } from '../lib/nav.svelte';
  import type { PostersData } from '../lib/types';
  import PosterCard from '../components/PosterCard.svelte';

  let { data }: { data: PostersData } = $props();

  // The link endpoint wants a form token at least a few seconds old (the same
  // minimum-time check as the submit form). Real visitors are always slower
  // than that; wait out the rest rather than be mistaken for a bot.
  const MIN_AGE_MS = 4500;
  // svelte-ignore state_referenced_locally (the first token comes with the page; later ones are fetched)
  let token = $state({ value: data?.form_token ?? '', issued: Date.now() });

  async function formToken(): Promise<string> {
    if (!token.value) token = { value: await freshFormToken(), issued: Date.now() };
    const wait = token.issued + MIN_AGE_MS - Date.now();
    if (wait > 0) await new Promise((r) => setTimeout(r, wait));
    const v = token.value;
    token = { value: '', issued: 0 }; // one use each; the next call fetches a fresh one
    return v;
  }
</script>

<div class="wrap posters">
  <h1>Shop posters</h1>
  <p class="lede">
    Free posters for your shop wall: Paul Bunyan and Babe on wood dust. Each one carries a QR code that opens the
    facts, and the sources behind them.
  </p>
  <p class="muted">
    Print on Letter or A4. Add your shop’s name on a “Posted by” plaque if you like; the name stays in your browser
    unless you choose to share it.
  </p>

  {#each data?.posters ?? [] as p (p.id)}
    <PosterCard poster={p} custom={data.custom} {formToken} />
  {:else}
    <p>No posters yet.</p>
  {/each}
</div>

<style>
  .posters {
    padding-top: 1.25rem;
  }

  .lede {
    max-width: 44rem;
  }
</style>
