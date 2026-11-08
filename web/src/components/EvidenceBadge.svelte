<script lang="ts">
  import { evidenceLabel } from '../lib/labels';

  let { level, short = false }: { level: number | null | undefined; short?: boolean } = $props();
  const ev = $derived(evidenceLabel(level));
</script>

<!-- Text is always shown; colour and icon are extra cues, never the only one. -->
<span class="badge ev{ev.level}" title={short ? ev.label : ev.explain}>
  <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true" focusable="false">
    {#if ev.level === 1}
      <!-- flask: clinical testing -->
      <path d="M6 1.5h4M6.5 1.5v4.2L2.6 12.6A1.3 1.3 0 0 0 3.7 14.5h8.6a1.3 1.3 0 0 0 1.1-1.9L9.5 5.7V1.5M4.6 10h6.8" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
    {:else if ev.level === 2}
      <!-- columns: agency -->
      <path d="M2 6 8 2.5 14 6M3 13.5h10M4 7.5v4.5M8 7.5v4.5M12 7.5v4.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
    {:else if ev.level === 3}
      <!-- speech bubble: reported -->
      <path d="M2.5 3.5h11v7h-6l-3 3v-3h-2z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
    {:else}
      <!-- tree rings: all wood -->
      <g fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="1.6" /><circle cx="8" cy="8" r="4" /><circle cx="8" cy="8" r="6.4" /></g>
    {/if}
  </svg>
  <span>{short ? ev.short : ev.label}</span>
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 0.35em;
    padding: 0.15em 0.6em;
    border-radius: 999px;
    font-size: 0.82rem;
    font-weight: 650;
    line-height: 1.5;
    white-space: nowrap;
    border: 1px solid currentColor;
  }

  .ev1 {
    background: var(--ev1-bg);
    color: var(--ev1-fg);
  }

  .ev2 {
    background: var(--ev2-bg);
    color: var(--ev2-fg);
  }

  .ev3 {
    background: var(--ev3-bg);
    color: var(--ev3-fg);
  }

  .ev0 {
    background: var(--ev0-bg);
    color: var(--ev0-fg);
    border-style: dashed;
  }

  svg {
    flex: none;
  }

  @media print {
    .badge {
      background: none;
      color: #000;
    }
  }
</style>
