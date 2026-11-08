<script lang="ts">
  import Art from '../components/Art.svelte';
  import SearchBox from '../components/SearchBox.svelte';
</script>

<div class="wrap nf">
  <!-- Art slot: images/mascots/babe-404.png (images/prompts/site-art.md §4). -->
  <Art
    src="/images/mascots/babe-404.png"
    alt="Babe the Blue Ox peeks around a tall stack of lumber, looking puzzled."
    width={320}
    height={320}
    eager
    class="nf-art"
  />
  <h1>We couldn't find that page</h1>
  <p class="muted">Babe looked behind every stack of lumber. Try searching for the wood instead.</p>
  <div class="search">
    <SearchBox variant="hero" id="nf-search" />
  </div>
  <p><a href="/">Home</a> · <a href="/browse">Browse all woods</a> · <a href="/protection">Protection</a></p>
</div>

<style>
  .nf {
    text-align: center;
    padding-top: 2rem;
    max-width: 40rem;
  }

  .nf :global(.nf-art) {
    width: min(100%, 260px);
    margin: 0 auto;
  }

  .search {
    margin: 1.5rem 0;
  }
</style>
