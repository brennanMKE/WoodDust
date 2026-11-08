<script lang="ts">
  // Copies text to the clipboard and announces the result. Falls back to
  // selecting the text (for the user to copy) where the Clipboard API is blocked.
  let {
    text,
    label = 'Copy',
    what = 'Text',
    selectTarget,
  }: { text: string; label?: string; what?: string; selectTarget?: HTMLElement | null } = $props();

  let message = $state('');
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function copy() {
    clearTimeout(timer);
    try {
      await navigator.clipboard.writeText(text);
      message = `${what} copied.`;
    } catch {
      if (selectTarget) {
        const range = document.createRange();
        range.selectNodeContents(selectTarget);
        const sel = getSelection();
        sel?.removeAllRanges();
        sel?.addRange(range);
      }
      message = `Couldn’t copy automatically. ${selectTarget ? 'The text is selected; ' : ''}press Ctrl+C or ⌘C to copy.`;
    }
    timer = setTimeout(() => (message = ''), 4000);
  }
</script>

<button type="button" class="button copy" onclick={copy}>
  <svg viewBox="0 0 20 20" width="16" height="16" aria-hidden="true" focusable="false">
    <rect x="7" y="7" width="10" height="10" rx="1.5" fill="none" stroke="currentColor" stroke-width="1.6" />
    <path d="M13 4.5V4a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h.5" fill="none" stroke="currentColor" stroke-width="1.6" />
  </svg>
  {label}
</button>
<span class="copy-status" role="status" aria-live="polite">{message}</span>

<style>
  .copy {
    min-height: 40px;
    padding: 0.35rem 0.8rem;
    font-size: 0.95rem;
  }

  .copy-status {
    font-size: 0.9rem;
    color: var(--muted);
    margin-left: 0.5rem;
  }
</style>
