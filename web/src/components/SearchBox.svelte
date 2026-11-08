<script lang="ts">
  import { asset } from '../lib/asset';
  // Editable combobox with list autocomplete (WAI-ARIA APG pattern).
  // Debounced type-ahead from /api/v1/search; Enter goes to the highlighted
  // option or the top match; no match shows "Not listed yet".
  // mode="pick" reuses it as a species picker inside another form: no <form>
  // of its own, and choosing calls onpick instead of navigating.
  import type { Snippet } from 'svelte';
  import { getJSON } from '../lib/api';
  import { comboInput, comboKey, initialCombo, type ComboState } from '../lib/combobox';
  import { navigate } from '../lib/nav.svelte';
  import type { SearchResponse, SearchResult } from '../lib/types';
  import Art from './Art.svelte';

  let {
    variant = 'hero',
    initialQuery = '',
    id = 'search',
    mode = 'navigate',
    label = 'Search by common, trade or botanical name',
    labelHidden = false,
    onpick,
    describedby,
    invalid = false,
    nomatch,
  }: {
    variant?: 'hero' | 'compact' | 'field';
    initialQuery?: string;
    id?: string;
    mode?: 'navigate' | 'pick';
    label?: string;
    /** Keep the label for assistive tech but don't show it (the placeholder carries the hint). */
    labelHidden?: boolean;
    onpick?: (r: SearchResult) => void;
    /** Extra ids for aria-describedby (hints, errors). */
    describedby?: string;
    invalid?: boolean;
    /** Replaces the "Not listed yet" text in pick mode; receives the query. */
    nomatch?: Snippet<[string]>;
  } = $props();
  const picking = $derived(mode === 'pick');

  const DEBOUNCE_MS = 150;
  const LIMIT = 8;

  // svelte-ignore state_referenced_locally
  let query = $state(initialQuery);
  let results: SearchResult[] = $state([]);
  let resultsFor = $state<string | null>(null);
  let combo: ComboState = $state({ ...initialCombo });
  let noMatch = $state(false);
  let loading = $state(false);
  let input: HTMLInputElement;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let controller: AbortController | null = null;

  const listId = $derived(`${id}-listbox`);
  const optionId = (i: number) => `${id}-opt-${i}`;
  const trimmed = $derived(query.trim());
  const showNoMatch = $derived(noMatch && trimmed.length > 0);

  const status = $derived.by(() => {
    if (!trimmed || loading) return '';
    if (showNoMatch) return `${trimmed} is not listed yet.`;
    if (combo.open && results.length)
      return `${results.length} suggestion${results.length === 1 ? '' : 's'}. Use up and down arrows to choose.`;
    return '';
  });

  async function runSearch(q: string): Promise<SearchResult[]> {
    controller?.abort();
    if (!q) {
      results = [];
      resultsFor = '';
      loading = false;
      return [];
    }
    const ctl = new AbortController();
    controller = ctl;
    loading = true;
    try {
      const res = await getJSON<SearchResponse>(
        `/api/v1/search?q=${encodeURIComponent(q)}&limit=${LIMIT}`,
        ctl.signal,
      );
      if (ctl.signal.aborted) return results;
      results = res.results ?? [];
      resultsFor = q;
      return results;
    } catch {
      if (!ctl.signal.aborted) {
        results = [];
        resultsFor = q;
      }
      return results;
    } finally {
      if (controller === ctl) loading = false;
    }
  }

  function oninput() {
    noMatch = false;
    clearTimeout(timer);
    const q = trimmed;
    timer = setTimeout(async () => {
      const r = await runSearch(q);
      if (q !== query.trim()) return;
      combo = comboInput(r.length);
      // Typed something we don't have: say so right away, without waiting for Enter.
      noMatch = q.length >= 2 && r.length === 0;
    }, DEBOUNCE_MS);
  }

  function go(r: SearchResult) {
    combo = { open: false, active: -1 };
    query = r.name;
    if (picking) onpick?.(r);
    else void navigate(r.url);
  }

  async function onkeydown(e: KeyboardEvent) {
    if (e.isComposing) return;
    let count = results.length;
    if (e.key === 'Enter') {
      e.preventDefault();
      if (!trimmed) return;
      if (resultsFor !== trimmed) {
        clearTimeout(timer);
        count = (await runSearch(trimmed)).length;
      }
    }
    const r = comboKey(combo, e.key, count);
    if (r.preventDefault) e.preventDefault();
    combo = r.state;
    switch (r.action.kind) {
      case 'select':
        go(results[r.action.index]);
        break;
      case 'no_match':
        noMatch = true;
        break;
      case 'clear':
        query = '';
        results = [];
        resultsFor = '';
        noMatch = false;
        break;
    }
  }

  function onsubmit(e: SubmitEvent) {
    e.preventDefault();
    void onkeydown(new KeyboardEvent('keydown', { key: 'Enter' }));
  }

  function onfocusout(e: FocusEvent) {
    const next = e.relatedTarget as Node | null;
    if (!next || !(e.currentTarget as HTMLElement).contains(next)) combo = { open: false, active: -1 };
  }

  // Prefill (e.g. /?q=…) runs the search once.
  $effect(() => {
    if (initialQuery.trim()) {
      void runSearch(initialQuery.trim()).then((r) => {
        noMatch = r.length === 0;
      });
    }
  });

  // Keep the highlighted option scrolled into view.
  $effect(() => {
    if (combo.open && combo.active >= 0) {
      document.getElementById(optionId(combo.active))?.scrollIntoView({ block: 'nearest' });
    }
  });
</script>

<svelte:element
  this={picking ? 'div' : 'form'}
  class="search {variant}"
  role={picking ? undefined : 'search'}
  onsubmit={picking ? undefined : onsubmit}
  {onfocusout}
  action={picking ? undefined : '/'}
  method={picking ? undefined : 'get'}
>
  <label
    for={id}
    class={variant === 'compact' || labelHidden ? 'visually-hidden' : variant === 'field' ? 'field-label' : 'label'}
  >
    {label}
  </label>
  <div class="field">
    <svg class="icon" viewBox="0 0 20 20" width="20" height="20" aria-hidden="true" focusable="false">
      <circle cx="8.5" cy="8.5" r="5.5" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="m13 13 4.5 4.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
    <input
      bind:this={input}
      {id}
      name={picking ? undefined : 'q'}
      type="text"
      role="combobox"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      enterkeyhint="go"
      placeholder={variant === 'compact' ? 'Search woods' : 'Oak, cocobolo, Thuja plicata…'}
      aria-invalid={invalid || undefined}
      aria-autocomplete="list"
      aria-expanded={combo.open && results.length > 0}
      aria-controls={listId}
      aria-activedescendant={combo.open && combo.active >= 0 ? optionId(combo.active) : undefined}
      aria-describedby={[showNoMatch ? `${id}-nomatch` : '', describedby ?? ''].filter(Boolean).join(' ') || undefined}
      bind:value={query}
      {oninput}
      {onkeydown}
      onfocus={() => {
        if (results.length && resultsFor === trimmed && trimmed) combo = { open: true, active: -1 };
      }}
    />
    {#if variant === 'hero'}
      <button class="go" type="submit">Search</button>
    {/if}
  </div>

  <ul id={listId} role="listbox" aria-label="Suggestions" class="listbox" hidden={!(combo.open && results.length > 0)}>
    {#each results as r, i (r.slug + r.matched_name)}
      <!-- Keyboard is handled on the input (aria-activedescendant); options are click targets only. -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <li
        id={optionId(i)}
        role="option"
        aria-selected={i === combo.active}
        class:active={i === combo.active}
        onmousedown={(e) => e.preventDefault()}
        onclick={() => go(r)}
      >
        <span class="opt-name">{r.name}</span>
        <span class="opt-meta">
          <span class="botanical">{r.botanical_name}</span>
          {#if r.matched_name && r.matched_name.toLowerCase() !== r.name.toLowerCase() && r.matched_name !== r.botanical_name}
            <span class="matched">· also called {r.matched_name}</span>
          {/if}
        </span>
      </li>
    {/each}
  </ul>

  {#if showNoMatch}
    <div class="nomatch" id="{id}-nomatch">
      {#if variant === 'hero'}
        <Art
          src={asset('/images/mascots/no-results.png')}
          alt="Paul Bunyan studies the end grain of a log through a large magnifying glass."
          width={160}
          height={160}
          fallback="hide"
          class="nomatch-art"
        />
      {/if}
      <div>
        {#if picking && nomatch}
          {@render nomatch(trimmed)}
        {:else}
          <p><strong>Not listed yet:</strong> “{trimmed}”.</p>
          <p class="muted">
            Check the spelling, try the botanical name, or
            <a href="/submit?kind=new_species&amp;name={encodeURIComponent(trimmed)}">suggest it</a>. Until then, treat its dust like
            any other: <a href="/protection">control it at the source and wear a respirator</a>.
          </p>
        {/if}
      </div>
    </div>
  {/if}

  <div class="visually-hidden" role="status" aria-live="polite">{status}</div>
</svelte:element>

<style>
  .search {
    position: relative;
    width: 100%;
  }

  .label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.4rem;
    color: var(--muted);
    text-align: center;
  }

  .field {
    position: relative;
    display: flex;
    align-items: center;
    background: var(--surface);
    border: 2px solid var(--border);
    border-radius: 999px;
    box-shadow: var(--shadow);
  }

  .field:focus-within {
    border-color: var(--blue);
  }

  .icon {
    position: absolute;
    left: 1rem;
    color: var(--muted);
    pointer-events: none;
  }

  input {
    flex: 1;
    min-width: 0;
    font: inherit;
    font-size: 1.15rem;
    color: var(--text);
    background: transparent;
    border: 0;
    padding: 0.85rem 1rem 0.85rem 2.9rem;
    border-radius: 999px;
    min-height: 52px;
  }

  input:focus-visible {
    outline: none;
  }

  input::placeholder {
    color: var(--muted);
    opacity: 0.85;
  }

  .go {
    font: inherit;
    font-weight: 700;
    margin: 4px;
    padding: 0 1.25rem;
    min-height: 44px;
    border-radius: 999px;
    border: 0;
    background: var(--accent);
    color: var(--on-accent);
    cursor: pointer;
  }

  .field-label {
    display: block;
    font-weight: 650;
    margin-bottom: 0.3rem;
  }

  .search.field > .field {
    border-radius: var(--radius-sm);
    box-shadow: none;
    border-width: 1px;
  }

  .search.field input {
    font-size: 1rem;
    border-radius: var(--radius-sm);
    min-height: 44px;
    padding: 0.5rem 0.75rem 0.5rem 2.6rem;
  }

  .search.field .icon {
    left: 0.8rem;
  }

  .search.field > .field:has(input[aria-invalid='true']) {
    border-color: var(--accent);
    border-width: 2px;
  }

  .compact .field {
    box-shadow: none;
    border-width: 1px;
  }

  .compact input {
    font-size: 1rem;
    padding: 0.45rem 0.75rem 0.45rem 2.5rem;
    min-height: 44px;
  }

  .compact .icon {
    left: 0.75rem;
  }

  .listbox {
    position: absolute;
    z-index: 30;
    left: 0;
    right: 0;
    margin: 0.35rem 0 0;
    padding: 0.35rem;
    list-style: none;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    max-height: 22rem;
    overflow-y: auto;
    text-align: left;
  }

  .listbox[hidden] {
    display: none;
  }

  li {
    display: flex;
    flex-direction: column;
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius-sm);
    cursor: pointer;
  }

  li:hover {
    background: var(--surface-2);
  }

  li.active {
    background: var(--blue);
    color: var(--on-blue);
  }

  li.active .opt-meta {
    color: inherit;
  }

  .opt-name {
    font-weight: 650;
  }

  .opt-meta {
    font-size: 0.88rem;
    color: var(--muted);
  }

  .compact .listbox {
    min-width: 18rem;
    right: auto;
  }

  .nomatch {
    display: flex;
    gap: 1rem;
    align-items: center;
    margin-top: 1rem;
    padding: 1rem;
    text-align: left;
    background: var(--surface);
    border: 1px dashed var(--border);
    border-radius: var(--radius);
  }

  .nomatch p {
    margin: 0 0 0.35rem;
  }

  .nomatch :global(.nomatch-art) {
    flex: none;
    width: 96px;
  }

  .compact .nomatch {
    position: absolute;
    z-index: 30;
    min-width: 18rem;
    box-shadow: var(--shadow);
  }

  @media (max-width: 30rem) {
    .go {
      padding: 0 0.9rem;
    }
  }
</style>
