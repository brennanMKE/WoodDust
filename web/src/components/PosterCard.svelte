<script lang="ts">
  // One poster on /posters: the sheet, its print files, and the "Posted by"
  // plaque builder (docs/posters.md). The shop name stays in the browser
  // unless the visitor ticks the opt-in; then it is sent once, as the
  // ShortLinks link's placement, never in a URL.
  import { tick } from 'svelte';
  import { app, navigate } from '../lib/nav.svelte';
  import { drafts } from '../lib/posterDraft.svelte';
  import {
    FormTokenExpired,
    SHOP_MAX,
    canvasToPNG,
    cleanShop,
    downloadBlob,
    fillTemplate,
    posterLabel,
    renderToCanvas,
    requestLink,
    slugFile,
  } from '../lib/poster';
  import type { PosterLink, PosterView } from '../lib/types';
  import PosterSheet from './PosterSheet.svelte';

  let {
    poster,
    custom,
    formToken,
  }: {
    poster: PosterView;
    /** The server can mint a plaque's own link. */
    custom: boolean;
    /** The page's form token, and a way to get a fresh one. */
    formToken: () => Promise<string>;
  } = $props();

  let building = $state(false);
  let shop = $state('');
  let share = $state(false);
  let website = $state(''); // honeypot
  let minted = $state<{ link: PosterLink; sharedAs: string } | null>(null);
  let busy = $state<'' | 'png' | 'pdf'>('');
  let notice = $state('');
  let error = $state('');

  const plaque = $derived(building ? cleanShop(shop) : null);
  const sharedName = $derived(share ? cleanShop(shop) : '');
  // A minted link stays valid for any name unless the name was shared with it.
  const link = $derived(minted && minted.sharedAs === sharedName ? minted.link : poster.stock);

  async function ensureLink(): Promise<PosterLink> {
    if (minted && minted.sharedAs === sharedName) return minted.link;
    if (!custom) return poster.stock;
    const req = { poster: poster.id, form_token: await formToken(), website, ...(sharedName ? { shop: sharedName } : {}) };
    let res;
    try {
      res = await requestLink(req, poster.stock);
    } catch (e) {
      if (!(e instanceof FormTokenExpired)) throw e;
      res = await requestLink({ ...req, form_token: await formToken() }, poster.stock);
    }
    notice = res.notice;
    minted = { link: res.link, sharedAs: sharedName };
    return res.link;
  }

  async function downloadPNG() {
    busy = 'png';
    error = '';
    try {
      const l = await ensureLink();
      const art = poster.art.available ? poster.art.url : null;
      const svg = fillTemplate(poster.template, { art, qrSVG: l.qr_svg, plaque }, posterLabel(poster, plaque));
      if (!svg) throw new Error('This poster can’t be drawn here.');
      const blob = await canvasToPNG(await renderToCanvas(svg, 2));
      downloadBlob(blob, slugFile(poster.id, plaque));
    } catch (e) {
      error = e instanceof Error ? e.message : 'Something went wrong. Try again.';
    } finally {
      busy = '';
    }
  }

  async function printPDF() {
    busy = 'pdf';
    error = '';
    try {
      const l = building ? await ensureLink() : poster.stock;
      drafts[poster.id] = { plaque, link: l, autoPrint: true };
      await navigate(`/posters/${poster.id}/print`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Something went wrong. Try again.';
    } finally {
      busy = '';
    }
  }

  async function openBuilder() {
    building = true;
    await tick();
    document.getElementById(`shop-${poster.id}`)?.focus();
  }
</script>

<article class="card" aria-labelledby="poster-{poster.id}-title">
  <div class="preview">
    <PosterSheet {poster} {link} {plaque} />
  </div>
  <div class="about">
    <h2 id="poster-{poster.id}-title">{poster.title}</h2>
    <p class="slogan">“{poster.slogan}”</p>
    <p>The QR code opens <a href={poster.destination}>{poster.destination}</a>, with the sources behind it.</p>

    {#if poster.downloads.length}
      <h3>Ready to print</h3>
      <ul class="downloads">
        {#each poster.downloads as d (d.url)}
          <li><a href={d.url} download>{d.label}</a></li>
        {/each}
      </ul>
    {:else}
      <p class="muted">Print-ready PDFs are on their way. Until then, print it from your browser.</p>
    {/if}

    {#if !building}
      <p class="actions">
        <button type="button" class="button" onclick={printPDF} disabled={busy !== ''}>Print or save as PDF</button>
        <button type="button" class="button primary" onclick={openBuilder}>Add a “Posted by” plaque</button>
      </p>
    {:else}
      <form class="builder" onsubmit={(e) => e.preventDefault()} aria-labelledby="plaque-{poster.id}">
        <h3 id="plaque-{poster.id}">“Posted by” plaque</h3>
        <div class="field">
          <label for="shop-{poster.id}">Your shop’s name <span class="muted">(optional)</span></label>
          <input
            id="shop-{poster.id}"
            type="text"
            bind:value={shop}
            maxlength={SHOP_MAX}
            autocomplete="organization"
            placeholder="Leave empty to write it in by hand"
            aria-describedby="shop-{poster.id}-hint"
          />
          <p class="hint" id="shop-{poster.id}-hint">
            The name is printed on your poster only. It stays in your browser and isn’t sent to us.
          </p>
        </div>
        {#if custom}
          <label class="check">
            <input type="checkbox" bind:checked={share} disabled={!cleanShop(shop)} aria-describedby="share-{poster.id}-hint" />
            <span>Also share the shop name with {app.site.name}</span>
          </label>
          <p class="hint" id="share-{poster.id}-hint">
            Off unless you tick it. If you do, the name is saved with this poster’s QR code link so we can see which
            shops put posters up. It is never part of the web address and never shown publicly.
          </p>
          <p class="hint">Each poster with a plaque gets its own QR code, so we can count how often it’s scanned.</p>
        {/if}
        <!-- Honeypot (contract §6): people never see or reach this; bots fill it in. -->
        <div class="hp" aria-hidden="true">
          <label for="website-{poster.id}">Website (leave this empty)</label>
          <input id="website-{poster.id}" type="text" tabindex="-1" autocomplete="off" bind:value={website} />
        </div>
        <p class="actions">
          <button type="button" class="button primary" onclick={printPDF} disabled={busy !== ''}>
            {busy === 'pdf' ? 'Preparing…' : 'Print or save as PDF'}
          </button>
          <button type="button" class="button" onclick={downloadPNG} disabled={busy !== ''}>
            {busy === 'png' ? 'Drawing…' : 'Download PNG'}
          </button>
          <button type="button" class="button link" onclick={() => (building = false)}>Remove plaque</button>
        </p>
      </form>
    {/if}
    {#if notice}<p class="note" role="status">{notice}</p>{/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  </div>
</article>

<style>
  .card {
    display: grid;
    grid-template-columns: minmax(0, 18rem) minmax(0, 1fr);
    gap: 1.25rem 1.75rem;
    align-items: start;
    padding: 1.25rem 0;
    border-top: 1px solid var(--rule);
  }

  @media (max-width: 40rem) {
    .card {
      grid-template-columns: 1fr;
    }

    .preview {
      max-width: 20rem;
    }
  }

  h2 {
    margin: 0 0 0.25rem;
  }

  h3 {
    font-size: 1.05rem;
    margin: 1rem 0 0.4rem;
  }

  .slogan {
    font-family: 'Alfa Slab One', var(--font-serif);
    font-size: 1.35rem;
    line-height: 1.25;
    margin: 0 0 0.6rem;
  }

  .downloads {
    margin: 0;
    padding-left: 1.2rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin: 1rem 0 0;
  }

  .builder {
    margin-top: 1rem;
    padding: 0.9rem 1rem;
    background: var(--surface);
    border: 1px solid var(--rule);
    border-radius: var(--radius);
  }

  .builder h3 {
    margin-top: 0;
  }

  .field label {
    display: block;
    font-weight: 650;
    margin-bottom: 0.3rem;
  }

  .field input {
    width: 100%;
    max-width: 24rem;
    font: inherit;
    padding: 0.45rem 0.6rem;
    min-height: 44px;
    color: var(--text);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }

  .hint {
    font-size: 0.9rem;
    color: var(--muted);
    margin: 0.3rem 0 0.6rem;
  }

  .check {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    font-weight: 600;
    min-height: 44px;
  }

  .check input {
    width: 1.15rem;
    height: 1.15rem;
    accent-color: var(--blue);
  }

  .button.link {
    background: none;
    border-color: transparent;
    text-decoration: underline;
  }

  .note {
    margin: 0.75rem 0 0;
    padding: 0.4rem 0.7rem;
    border-left: 3px solid var(--wood);
    background: var(--ev3-bg);
    color: var(--ev3-fg);
  }

  .error {
    color: var(--accent);
    font-weight: 600;
    margin: 0.75rem 0 0;
  }

  .hp {
    position: absolute;
    left: -10000px;
    top: auto;
    width: 1px;
    height: 1px;
    overflow: hidden;
  }
</style>
