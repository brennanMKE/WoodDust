<script lang="ts">
  // An illustration slot. Files and alt text are defined in
  // images/prompts/site-art.md; the site picks them up by filename.
  // If the file is missing (or fails to load) the slot either keeps its box
  // with a faint tree-ring drawing ("rings", no layout shift) or disappears
  // ("hide", for optional decoration).
  let {
    src,
    alt,
    width,
    height,
    eager = false,
    fallback = 'rings',
    class: className = '',
  }: {
    src: string;
    alt: string;
    width: number;
    height: number;
    eager?: boolean;
    fallback?: 'rings' | 'hide';
    class?: string;
  } = $props();

  let failed = $state(false);
</script>

{#if !(failed && fallback === 'hide')}
  <div class="art {className}" style:aspect-ratio="{width} / {height}">
    {#if !failed}
      <img
        {src}
        {alt}
        {width}
        {height}
        loading={eager ? 'eager' : 'lazy'}
        decoding="async"
        onerror={() => (failed = true)}
      />
    {:else}
      <svg viewBox="0 0 120 120" aria-hidden="true" focusable="false">
        <g fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
          <ellipse cx="60" cy="60" rx="8" ry="7" />
          <ellipse cx="60" cy="60" rx="20" ry="18" opacity=".85" />
          <ellipse cx="60" cy="60" rx="33" ry="30" opacity=".7" />
          <ellipse cx="60" cy="60" rx="46" ry="43" opacity=".55" />
          <path d="M60 60 L104 46" opacity=".55" />
        </g>
      </svg>
    {/if}
  </div>
{/if}

<style>
  .art {
    display: grid;
    place-items: center;
    width: 100%;
    color: var(--wood);
  }

  img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }

  svg {
    width: 60%;
    height: 60%;
    opacity: 0.4;
  }
</style>
