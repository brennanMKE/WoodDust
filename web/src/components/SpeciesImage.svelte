<script lang="ts">
  import type { Image } from '../lib/types';

  let { image, eager = false, sizes = '100vw' }: { image: Image; eager?: boolean; sizes?: string } = $props();
  const PLACEHOLDER = '/images/placeholder.svg';
  // 0: the image itself, 1: the placeholder SVG, 2: just the wood-tone frame.
  let stage = $state(0);
  const src = $derived(stage === 0 && image?.url ? image.url : PLACEHOLDER);
  const failed = $derived(stage >= 2);

  function onerror() {
    stage = src === PLACEHOLDER ? 2 : 1;
  }
</script>

<!-- Fixed 1200×630 frame with a wood-tone backdrop, so a slow or missing image never shifts layout. -->
<div class="frame">
  {#if !failed}
    <img
      {src}
      alt={image?.alt ?? ''}
      width={image?.width ?? 1200}
      height={image?.height ?? 630}
      loading={eager ? 'eager' : 'lazy'}
      fetchpriority={eager ? 'high' : 'auto'}
      decoding="async"
      {sizes}
      {onerror}
    />
  {/if}
</div>

<style>
  .frame {
    aspect-ratio: 1200 / 630;
    width: 100%;
    overflow: hidden;
    background:
      repeating-radial-gradient(ellipse at 30% 140%, transparent 0 10px, rgb(0 0 0 / 0.05) 10px 12px),
      linear-gradient(135deg, #c9a37a, #9c6b3f);
  }

  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
</style>
