<script lang="ts">
  import { tick } from 'svelte';
  import { app } from './lib/nav.svelte';
  import Header from './components/Header.svelte';
  import Footer from './components/Footer.svelte';
  import Home from './views/Home.svelte';
  import Species from './views/Species.svelte';
  import Browse from './views/Browse.svelte';
  import Page from './views/Page.svelte';
  import Poster from './views/Poster.svelte';
  import Sources from './views/Sources.svelte';
  import Submit from './views/Submit.svelte';
  import SubmissionStatus from './views/SubmissionStatus.svelte';
  import NotFound from './views/NotFound.svelte';
  import type {
    ContentPage,
    MeResponse,
    SourcesResponse,
    SpeciesDetail,
    SpeciesList,
    SubmissionStatusData,
    SubmitData,
  } from './lib/types';

  // Admin is owner-only: keep it in its own chunk, loaded on first visit.
  let adminView: Promise<typeof import('./admin/Admin.svelte')> | null = null;
  const loadAdmin = () => (adminView ??= import('./admin/Admin.svelte'));

  // /submit?kind=… and /submit?name=… re-initialize the form, so key on the query too.
  const viewKey = $derived(app.match.id === 'submit' ? app.match.path + app.search : app.match.path);

  let announcement = $state('');

  // After a client-side navigation, move focus to the new page's heading and
  // announce it, so keyboard and screen-reader users know the page changed.
  $effect(() => {
    if (app.navCount === 0) return;
    void tick().then(() => {
      const h1 = document.querySelector<HTMLElement>('#main h1');
      if (h1) {
        h1.setAttribute('tabindex', '-1');
        h1.focus({ preventScroll: true });
      }
      announcement = document.title;
    });
  });
</script>

<a class="skip-link" href="#main">Skip to content</a>
<Header />

<main id="main" tabindex="-1" aria-busy={app.loading}>
  {#if app.error}
    <div class="wrap">
      <div class="notice" role="alert">
        <p><strong>That page didn't load.</strong> {app.error} Check your connection and try again.</p>
      </div>
    </div>
  {/if}
  {#if app.loading}
    <div class="progress" aria-hidden="true"></div>
  {/if}

  {#key viewKey}
    {#if app.match.id === 'home'}
      <Home data={app.data as SpeciesList} />
    {:else if app.match.id === 'species'}
      <Species species={app.data as SpeciesDetail} />
    {:else if app.match.id === 'browse'}
      <Browse data={app.data as SpeciesList} />
    {:else if app.match.id === 'page' && app.match.params.page?.startsWith('print/')}
      <Poster page={app.data as ContentPage} />
    {:else if app.match.id === 'page'}
      <Page page={app.data as ContentPage} />
    {:else if app.match.id === 'sources'}
      <Sources data={app.data as SourcesResponse} />
    {:else if app.match.id === 'submit'}
      <Submit slug={app.match.params.slug} data={app.data as SubmitData | null} />
    {:else if app.match.id === 'submission_status'}
      <SubmissionStatus data={app.data as SubmissionStatusData} />
    {:else if app.match.id === 'admin'}
      {#await loadAdmin()}
        <div class="wrap"><p class="muted">Loading admin…</p></div>
      {:then { default: Admin }}
        <Admin data={app.data as MeResponse | null} section={app.match.params.section} id={app.match.params.id} />
      {:catch}
        <div class="wrap"><div class="notice" role="alert"><p>The admin screens didn’t load. Reload the page to try again.</p></div></div>
      {/await}
    {:else}
      <NotFound />
    {/if}
  {/key}
</main>

<Footer />
<div class="visually-hidden" aria-live="polite" aria-atomic="true">{announcement}</div>

<style>
  main {
    display: block;
    min-height: 60vh;
    padding-bottom: 3rem;
  }

  main:focus {
    outline: none;
  }

  @media print {
    main {
      min-height: 0;
      padding-bottom: 0;
    }
  }

  .progress {
    position: fixed;
    top: 0;
    left: 0;
    height: 3px;
    width: 100%;
    background: linear-gradient(90deg, var(--accent), var(--blue));
    transform-origin: left;
    animation: grow 1.2s ease-out infinite;
    z-index: 50;
  }

  @keyframes grow {
    from {
      transform: scaleX(0);
    }
    to {
      transform: scaleX(1);
    }
  }
</style>
