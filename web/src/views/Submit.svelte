<script lang="ts">
  import { asset } from '../lib/asset';
  // Anonymous submission form (docs/contract.md §6, PLAN.md §6). The rules live
  // in lib/submission.ts (a tested reducer); this view wires it to the DOM.
  import { tick } from 'svelte';
  import { app, freshFormToken } from '../lib/nav.svelte';
  import { HAZARD_NAMES, HAZARD_TYPES } from '../lib/labels';
  import { guessCitationType, leadsOnlyWarning, type CitationType } from '../lib/citations';
  import {
    DETAILS_MAX,
    DETAILS_MIN,
    KINDS,
    MAX_CITATIONS,
    charCount,
    friendlyWait,
    initialForm,
    isKind,
    needsCitation,
    parseRetryAfter,
    reduce,
    toPayload,
    warningTexts,
    type FormAction,
    type SubmissionKind,
    type TextField,
  } from '../lib/submission';
  import type { SearchResult, SubmitData } from '../lib/types';
  import Art from '../components/Art.svelte';
  import CopyButton from '../components/CopyButton.svelte';
  import SearchBox from '../components/SearchBox.svelte';

  let { slug, data }: { slug?: string; data: SubmitData | null } = $props();

  // svelte-ignore state_referenced_locally
  const params = new URLSearchParams(app.search);
  // svelte-ignore state_referenced_locally
  const initOpts = {
    kind: params.get('kind'),
    target: data?.target ?? null,
    name: data?.prefill_name || params.get('name') || '',
  };
  // svelte-ignore state_referenced_locally
  let formToken = $state(data?.form_token ?? '');
  let form = $state(initialForm(initOpts));
  let summaryEl: HTMLElement | undefined = $state();
  let bannerEl: HTMLElement | undefined = $state();
  let doneEl: HTMLElement | undefined = $state();
  let pickerKey = $state(0);

  function dispatch(a: FormAction) {
    form = reduce(form, a);
  }

  // Kinds the server offers (if it says), in our order.
  const kinds = $derived.by(() => {
    const offered = (data?.kinds ?? [])
      .map((k) => (typeof k === 'string' ? k : (k as { id?: string; kind?: string })?.id ?? (k as { kind?: string })?.kind))
      .filter(isKind);
    return offered.length ? KINDS.filter((k) => offered.includes(k.id)) : KINDS;
  });

  const missingSlug = $derived(Boolean(slug) && !data?.target);
  const detailsCount = $derived(charCount(form.details));
  const detailsLeft = $derived(DETAILS_MAX - detailsCount);
  // Announce the counter only at a few thresholds so typing isn't drowned out.
  const countAnnouncement = $derived(
    detailsLeft < 0
      ? `${-detailsLeft} characters over the limit.`
      : detailsLeft <= 100
        ? 'Fewer than 100 characters left.'
        : detailsLeft <= 500
          ? 'Fewer than 500 characters left.'
          : '',
  );
  const errorList = $derived(Object.entries(form.errors));

  const DETAILS_LABEL: Record<SubmissionKind, { label: string; hint: string }> = {
    new_species: {
      label: 'What does your source say about this wood’s dust?',
      hint: 'Name the hazard (for example asthma or a skin rash), what kind of study or listing reported it, and anything else we should check.',
    },
    correction: {
      label: 'What’s wrong, and what should it say?',
      hint: 'Tell us which part of the card is wrong and what the source says instead.',
    },
    new_source: {
      label: 'Which claim does this source support?',
      hint: 'For example: “the asthma entry: this is a challenge-tested case report”.',
    },
    reaction_report: {
      label: 'What happened?',
      hint: 'Which wood, what you were doing (sanding, turning, carving), the symptoms and how soon they started. Don’t include your name or anything else that identifies you.',
    },
  };

  const TYPE_LABEL: Record<CitationType, { label: string; placeholder: string; mode: 'text' | 'numeric' | 'url' }> = {
    doi: { label: 'DOI', placeholder: '10.1136/oem.48.3.201', mode: 'text' },
    pmid: { label: 'PubMed ID', placeholder: '2021399', mode: 'numeric' },
    url: { label: 'Web address', placeholder: 'https://www.osha.gov/…', mode: 'url' },
  };

  /** Field key -> element id, for the error summary links. */
  function fieldId(key: string): string | null {
    if (key === 'kind') return `kind-${kinds[0].id}`;
    if (key === 'target') return form.target ? 'target-change' : 'target-picker';
    if (key === 'species_name') return 'species-name';
    if (key === 'botanical_name') return 'botanical-name';
    if (key === 'hazard_type') return 'hazard-type';
    if (key === 'details') return 'details';
    if (key === 'email') return 'email';
    if (key === 'citations') return form.citations[0] ? `cite-value-${form.citations[0].id}` : null;
    const m = key.match(/^citations\.(\d+)$/);
    if (m) {
      const row = form.citations[Number(m[1])];
      return row ? `cite-value-${row.id}` : null;
    }
    return null;
  }

  function errorLabel(key: string): string {
    const m = key.match(/^citations\.(\d+)$/);
    if (m) return `Source ${Number(m[1]) + 1}`;
    return (
      {
        kind: 'Type of submission',
        target: 'Species',
        species_name: 'Common name',
        botanical_name: 'Botanical name',
        hazard_type: 'Hazard',
        details: 'Details',
        email: 'Email',
        citations: 'Sources',
        form_token: 'Form',
        form: 'Form',
      }[key] ?? key.replace(/_/g, ' ')
    );
  }

  function focusField(e: MouseEvent, key: string) {
    const id = fieldId(key);
    const el = id ? document.getElementById(id) : null;
    if (!el) return;
    e.preventDefault();
    el.scrollIntoView({ block: 'center' });
    el.focus();
  }

  function setText(field: TextField) {
    return (e: Event) => dispatch({ type: 'set', field, value: (e.currentTarget as HTMLInputElement).value });
  }

  function chooseKind(kind: SubmissionKind) {
    dispatch({ type: 'set_kind', kind });
  }

  function pick(r: SearchResult) {
    dispatch({ type: 'set_target', target: { slug: r.slug, name: r.name, botanical_name: r.botanical_name } });
    void tick().then(() => document.getElementById('target-change')?.focus());
  }

  async function changeTarget() {
    dispatch({ type: 'set_target', target: null });
    pickerKey++;
    await tick();
    document.getElementById('target-picker')?.focus();
  }

  function switchToNewSpecies(name: string) {
    dispatch({ type: 'set_kind', kind: 'new_species' });
    dispatch({ type: 'set', field: 'speciesName', value: name });
    void tick().then(() => document.getElementById('species-name')?.focus());
  }

  async function addCitation() {
    dispatch({ type: 'add_citation' });
    await tick();
    const last = form.citations[form.citations.length - 1];
    document.getElementById(`cite-type-${last.id}`)?.focus();
  }

  async function removeCitation(id: number, index: number) {
    dispatch({ type: 'remove_citation', id });
    await tick();
    const next = form.citations[Math.min(index, form.citations.length - 1)];
    document.getElementById(next ? `cite-value-${next.id}` : 'add-citation')?.focus();
  }

  let checkSeq = 0;
  async function checkCitation(id: number) {
    const row = form.citations.find((c) => c.id === id);
    if (!row) return;
    // A DOI or PMID pasted into the wrong box: switch the type for them.
    const guess = guessCitationType(row.value);
    if (guess && guess !== row.type && (guess !== 'url' || row.type === 'url')) {
      dispatch({ type: 'update_citation', id, patch: { type: guess } });
    }
    dispatch({ type: 'check_citation', id });
    const current = form.citations.find((c) => c.id === id)!;
    if (current.type !== 'url' || !current.value.trim()) return;
    const url = current.value.trim();
    const local = leadsOnlyWarning(url);
    dispatch({ type: 'url_warning', id, warning: local });
    const seq = ++checkSeq;
    try {
      const res = await fetch(`/api/v1/submissions/check-url?url=${encodeURIComponent(url)}`, {
        headers: { Accept: 'application/json' },
      });
      if (!res.ok || seq !== checkSeq) return;
      const body = (await res.json()) as { warnings?: unknown; warning?: string; leads_only?: boolean };
      const texts = [...warningTexts(body.warnings), ...(body.warning ? [body.warning] : [])];
      const still = form.citations.find((c) => c.id === id);
      if (!still || still.value.trim() !== url) return;
      if (texts.length) dispatch({ type: 'url_warning', id, warning: texts.join(' ') });
      else if (body.leads_only) dispatch({ type: 'url_warning', id, warning: local ?? 'This site is for leads only. Please also give the study or agency page it relies on.' });
      else if (!local) dispatch({ type: 'url_warning', id, warning: null });
    } catch {
      /* offline: the local check stands */
    }
  }

  async function onsubmit(e: SubmitEvent) {
    e.preventDefault();
    if (form.phase === 'submitting') return;
    dispatch({ type: 'validate' });
    if (Object.keys(form.errors).length) {
      await tick();
      summaryEl?.focus();
      return;
    }
    dispatch({ type: 'submit_start' });
    let res: Response;
    try {
      res = await fetch('/api/v1/submissions', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(toPayload(form, formToken)),
      });
    } catch {
      dispatch({ type: 'submit_failed', message: 'We couldn’t reach the server. Check your connection and send it again; nothing was lost.' });
      await tick();
      bannerEl?.focus();
      return;
    }
    let body: any = null;
    try {
      body = await res.json();
    } catch {
      /* empty or not JSON */
    }
    if (res.ok) {
      dispatch({
        type: 'submit_ok',
        result: {
          id: body?.id,
          reference: body?.reference ?? '',
          statusUrl: body?.status_url ?? '',
          warnings: warningTexts(body?.warnings),
        },
      });
      await tick();
      doneEl?.focus();
      window.scrollTo(0, 0);
      return;
    }
    if (res.status === 422) {
      const fields: Record<string, string> = { ...(body?.error?.fields ?? {}) };
      if (fields.form_token || body?.error?.code === 'form_expired') {
        // The signed token is old or invalid: get a new one, ask them to send again.
        delete fields.form_token;
        try {
          formToken = await freshFormToken();
          fields.form = 'The form had been open too long, so we refreshed it. Please check your answers and send it again.';
        } catch {
          fields.form = 'The form has expired. Reload the page and send it again.';
        }
      }
      dispatch({ type: 'submit_invalid', message: body?.error?.message ?? '', fields });
      await tick();
      summaryEl?.focus();
      return;
    }
    if (res.status === 429) {
      dispatch({ type: 'submit_rate_limited', retryAfter: parseRetryAfter(res.headers.get('Retry-After')) });
    } else {
      dispatch({
        type: 'submit_failed',
        message: body?.error?.message ? `${body.error.message}` : `The server had a problem (error ${res.status}). Please try again in a few minutes.`,
      });
    }
    await tick();
    bannerEl?.focus();
  }

  async function sendAnother() {
    try {
      formToken = await freshFormToken();
    } catch {
      /* keep the old one; the server will say if it's stale */
    }
    dispatch({ type: 'reset', opts: { kind: form.kind, target: form.target } });
    await tick();
    document.querySelector<HTMLElement>('#main h1')?.focus();
  }

  const statusHref = $derived(form.result?.statusUrl ?? '');
  const statusAbsolute = $derived(statusHref ? new URL(statusHref, location.origin).href : '');
  let statusLinkEl: HTMLElement | undefined = $state();

  const err = (key: string) => form.errors[key];
  const describe = (...ids: (string | false | undefined)[]) => ids.filter(Boolean).join(' ') || undefined;
</script>

<div class="wrap submit">
  <header class="intro">
    <Art
      src={asset('/images/art/spot-almanac.png')}
      alt="An open almanac with a ribbon bookmark and a wood shaving."
      width={160}
      height={160}
      fallback="hide"
      class="intro-art"
    />
    <div>
      <p class="eyebrow">Anonymous · manually reviewed</p>
      <h1>
        {#if form.phase === 'done'}Thank you{:else if form.target && form.kind === 'correction'}Report a correction{:else if form.kind === 'new_species'}Suggest a species{:else}Suggest or correct{/if}
      </h1>
      {#if form.phase !== 'done'}
        <p class="lede">
          Help keep the almanac honest. No account needed. Every submission is manually reviewed and checked
          against its sources before anything changes.
        </p>
      {/if}
    </div>
  </header>

  {#if form.phase === 'done' && form.result}
    <section class="done" aria-labelledby="done-title">
      <h2 id="done-title" tabindex="-1" bind:this={doneEl}>We’ve got it</h2>
      <p>Your reference is <strong class="ref">{form.result.reference}</strong>.</p>
      {#if statusHref}
        <div class="save">
          <p><strong>Save this link.</strong> It’s the only way to see your submission’s status and any questions we have. We can’t look it up for you, and it isn’t sent anywhere.</p>
          <p class="status-link">
            <a href={statusHref} bind:this={statusLinkEl}>{statusAbsolute}</a>
          </p>
          <CopyButton text={statusAbsolute} label="Copy link" what="Link" selectTarget={statusLinkEl} />
        </div>
      {/if}
      {#if form.result.warnings.length}
        <div class="warn" role="note">
          <p><strong>Please note:</strong></p>
          <ul>
            {#each form.result.warnings as w, i (i)}<li>{w}</li>{/each}
          </ul>
        </div>
      {/if}
      {#if form.kind === 'reaction_report'}
        <p class="muted">Reaction reports are never published. If you still have symptoms, please see a doctor.</p>
      {/if}
      <p class="next">
        <button type="button" class="button" onclick={sendAnother}>Send another</button>
        {#if form.target}<a href="/wood/{form.target.slug}">Back to {form.target.name}</a>{:else}<a href="/">Back to search</a>{/if}
      </p>
    </section>
  {:else}
    {#if missingSlug}
      <div class="notice" role="note">
        <p>We couldn’t find a listed species called “{slug}”. Choose one below, or suggest it as a new species.</p>
      </div>
    {/if}

    {#if form.banner}
      <div class="banner" role="alert" tabindex="-1" bind:this={bannerEl}>
        {#if form.banner.kind === 'rate_limited'}
          <p>
            <strong>We’re getting a lot of submissions right now.</strong> Please try again
            {friendlyWait(form.banner.retryAfter)}. Your answers are still here, so you don’t need to retype them.
          </p>
        {:else}
          <p><strong>That didn’t send.</strong> {form.banner.message}</p>
        {/if}
      </div>
    {/if}

    {#if errorList.length}
      <div class="error-summary" role="alert" aria-labelledby="error-summary-title" tabindex="-1" bind:this={summaryEl}>
        <h2 id="error-summary-title">
          {errorList.length === 1 ? 'One thing needs fixing' : `${errorList.length} things need fixing`}
        </h2>
        <ul>
          {#each errorList as [key, message] (key)}
            <li>
              {#if fieldId(key)}
                <a href="#{fieldId(key)}" onclick={(e) => focusField(e, key)}>{errorLabel(key)}: {message}</a>
              {:else}
                {message}
              {/if}
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <form novalidate {onsubmit} aria-busy={form.phase === 'submitting'}>
      <fieldset class="kinds" aria-describedby={err('kind') ? 'kind-error' : undefined}>
        <legend>What would you like to send?</legend>
        {#if err('kind')}<p class="field-error" id="kind-error">{err('kind')}</p>{/if}
        <div class="kind-grid">
          {#each kinds as k (k.id)}
            <label class="kind" class:selected={form.kind === k.id} class:reaction={k.id === 'reaction_report'}>
              <input
                type="radio"
                name="kind"
                id="kind-{k.id}"
                value={k.id}
                checked={form.kind === k.id}
                aria-describedby="kind-{k.id}-hint"
                onchange={() => chooseKind(k.id)}
              />
              <span class="kind-text">
                <span class="kind-label">{k.label}</span>
                <span class="kind-hint" id="kind-{k.id}-hint">{k.hint}</span>
              </span>
            </label>
          {/each}
        </div>
      </fieldset>

      {#if form.kind === 'reaction_report'}
        <section class="medical" aria-labelledby="medical-title">
          <h2 id="medical-title">This is not a medical service</h2>
          <p>
            We can’t give medical advice and nobody here will reply about your health. <strong>If you have symptoms, see a
            doctor</strong> and tell them which woods you work with. If you have trouble breathing, chest tightness or
            swelling of the face or throat, call your local emergency number now.
          </p>
          <p>
            Reaction reports are never published. They only help us decide which woods to research next.
          </p>
        </section>
      {/if}

      {#if form.kind}
        <fieldset class="group">
          <legend>{form.kind === 'new_species' ? 'The wood' : 'Which species?'}</legend>

          {#if form.kind === 'new_species'}
            <div class="row two">
              <div class="field">
                <label for="species-name">Common name <span class="req">(required)</span></label>
                <input
                  id="species-name"
                  type="text"
                  autocomplete="off"
                  value={form.speciesName}
                  oninput={setText('speciesName')}
                  aria-invalid={Boolean(err('species_name')) || undefined}
                  aria-describedby={describe(err('species_name') && 'species-name-error')}
                  maxlength="120"
                />
                {#if err('species_name')}<p class="field-error" id="species-name-error">{err('species_name')}</p>{/if}
              </div>
              <div class="field">
                <label for="botanical-name">Botanical name <span class="opt">(if you know it)</span></label>
                <input
                  id="botanical-name"
                  type="text"
                  autocomplete="off"
                  class="botanical"
                  placeholder="Guibourtia spp."
                  value={form.botanicalName}
                  oninput={setText('botanicalName')}
                  aria-invalid={Boolean(err('botanical_name')) || undefined}
                  aria-describedby={describe(err('botanical_name') && 'botanical-name-error')}
                  maxlength="160"
                />
                {#if err('botanical_name')}<p class="field-error" id="botanical-name-error">{err('botanical_name')}</p>{/if}
              </div>
            </div>
          {:else if form.target}
            <div class="chosen">
              <p>
                <span class="chosen-name">{form.target.name}</span>
                {#if form.target.botanical_name}<span class="botanical muted">{form.target.botanical_name}</span>{/if}
              </p>
              <button type="button" class="button" id="target-change" onclick={changeTarget}>
                Change<span class="visually-hidden"> species (currently {form.target.name})</span>
              </button>
            </div>
          {:else}
            {#key pickerKey}
              <SearchBox
                variant="field"
                mode="pick"
                id="target-picker"
                label={form.kind === 'reaction_report' ? 'Find the wood' : 'Find the species'}
                onpick={pick}
                invalid={Boolean(err('target'))}
                describedby={describe('target-hint', err('target') && 'target-error')}
                initialQuery={missingSlug && slug ? slug.replace(/-/g, ' ') : ''}
              >
                {#snippet nomatch(q: string)}
                  {#if form.kind === 'reaction_report'}
                    <p>No listed species matches “{q}”. That’s fine: type its name in the box below.</p>
                  {:else}
                    <p><strong>Not listed:</strong> “{q}”.</p>
                    <p>
                      <button type="button" class="linkish" onclick={() => switchToNewSpecies(q)}>
                        Suggest “{q}” as a new species instead
                      </button>
                    </p>
                  {/if}
                {/snippet}
              </SearchBox>
            {/key}
            <p class="hint" id="target-hint">Type a common, trade or botanical name, then choose from the list.</p>
            {#if err('target')}<p class="field-error" id="target-error">{err('target')}</p>{/if}
            {#if form.kind === 'reaction_report'}
              <div class="field">
                <label for="species-name">Not listed? Type the wood’s name <span class="opt">(optional)</span></label>
                <input
                  id="species-name"
                  type="text"
                  autocomplete="off"
                  value={form.speciesName}
                  oninput={setText('speciesName')}
                  maxlength="120"
                />
              </div>
            {/if}
          {/if}
        </fieldset>

        <fieldset class="group">
          <legend>{DETAILS_LABEL[form.kind].label}</legend>
          <div class="field">
            <label for="hazard-type">
              {form.kind === 'reaction_report' ? 'Closest kind of reaction' : 'Hazard'} <span class="opt">(optional)</span>
            </label>
            <select id="hazard-type" value={form.hazardType} onchange={setText('hazardType')}>
              <option value="">{form.kind === 'reaction_report' ? 'Not sure' : 'Not about one hazard / not sure'}</option>
              {#each HAZARD_TYPES as h (h)}<option value={h}>{HAZARD_NAMES[h]}</option>{/each}
            </select>
          </div>

          <div class="field">
            <label for="details">Details <span class="req">(required)</span></label>
            <p class="hint" id="details-hint">{DETAILS_LABEL[form.kind].hint}</p>
            <p class="own-words" id="details-words">
              <strong>In your own words, please.</strong> Don’t paste abstracts, paper text or charts from other sites:
              we can only use facts, and we write every description ourselves.
            </p>
            <textarea
              id="details"
              rows="7"
              value={form.details}
              oninput={setText('details')}
              aria-invalid={Boolean(err('details')) || undefined}
              aria-describedby={describe('details-hint', 'details-words', 'details-count', err('details') && 'details-error')}
            ></textarea>
            <p class="counter" id="details-count" class:over={detailsLeft < 0}>
              {detailsCount} of {DETAILS_MAX} characters{detailsCount < DETAILS_MIN ? ` (at least ${DETAILS_MIN})` : ''}
            </p>
            <p class="visually-hidden" aria-live="polite">{countAnnouncement}</p>
            {#if err('details')}<p class="field-error" id="details-error">{err('details')}</p>{/if}
          </div>
        </fieldset>

        <fieldset class="group" aria-describedby={describe('cite-hint', err('citations') && 'cite-error')}>
          <legend>
            Sources {#if needsCitation(form.kind)}<span class="req">(at least one)</span>{:else}<span class="opt">(optional)</span>{/if}
          </legend>
          <p class="hint" id="cite-hint">
            A DOI, a PubMed ID or the address of an official page (an agency such as OSHA, NIOSH, HSE or IARC). Sites
            like the Wood Database, Haz-Map and forums are useful leads, but we need the study or listing behind them.
          </p>
          {#if err('citations')}<p class="field-error" id="cite-error">{err('citations')}</p>{/if}
          <ol class="cites">
            {#each form.citations as c, i (c.id)}
              {@const rowErr = err(`citations.${i}`)}
              {@const warn = form.urlWarnings[c.id]}
              <li class="cite">
                <div class="cite-grid">
                  <div class="field">
                    <label for="cite-type-{c.id}">Source {i + 1} type</label>
                    <select
                      id="cite-type-{c.id}"
                      value={c.type}
                      onchange={(e) => dispatch({ type: 'update_citation', id: c.id, patch: { type: e.currentTarget.value as CitationType } })}
                    >
                      <option value="doi">DOI</option>
                      <option value="pmid">PubMed ID</option>
                      <option value="url">Web address</option>
                    </select>
                  </div>
                  <div class="field">
                    <label for="cite-value-{c.id}">{TYPE_LABEL[c.type].label}</label>
                    <input
                      id="cite-value-{c.id}"
                      type={c.type === 'url' ? 'url' : 'text'}
                      inputmode={TYPE_LABEL[c.type].mode}
                      autocomplete="off"
                      spellcheck="false"
                      placeholder={TYPE_LABEL[c.type].placeholder}
                      value={c.value}
                      oninput={(e) => dispatch({ type: 'update_citation', id: c.id, patch: { value: e.currentTarget.value } })}
                      onblur={() => checkCitation(c.id)}
                      aria-invalid={Boolean(rowErr) || undefined}
                      aria-describedby={describe(rowErr && `cite-error-${c.id}`, warn && `cite-warn-${c.id}`)}
                      maxlength="500"
                    />
                  </div>
                </div>
                {#if rowErr}<p class="field-error" id="cite-error-{c.id}">{rowErr}</p>{/if}
                {#if warn}<p class="field-warn" id="cite-warn-{c.id}" role="status">{warn}</p>{/if}
                <div class="field">
                  <label for="cite-text-{c.id}">Note about source {i + 1} <span class="opt">(optional: author, year, title, page)</span></label>
                  <input
                    id="cite-text-{c.id}"
                    type="text"
                    autocomplete="off"
                    value={c.text}
                    oninput={(e) => dispatch({ type: 'update_citation', id: c.id, patch: { text: e.currentTarget.value } })}
                    maxlength="300"
                  />
                </div>
                {#if form.citations.length > 1 || c.value || c.text}
                  <button type="button" class="linkish remove" onclick={() => removeCitation(c.id, i)}>
                    Remove source {i + 1}
                  </button>
                {/if}
              </li>
            {/each}
          </ol>
          {#if form.citations.length < MAX_CITATIONS}
            <button type="button" class="button" id="add-citation" onclick={addCitation}>+ Add another source</button>
          {/if}
        </fieldset>

        <fieldset class="group">
          <legend>Follow-up <span class="opt">(optional)</span></legend>
          <div class="field">
            <label for="email">Email</label>
            <p class="hint" id="email-hint">
              Only if you don’t mind a follow-up question. It’s never shown publicly or shared, and we delete it when your
              submission is closed. Leave it empty to stay fully anonymous.
            </p>
            <input
              id="email"
              type="email"
              autocomplete="email"
              value={form.email}
              oninput={setText('email')}
              aria-invalid={Boolean(err('email')) || undefined}
              aria-describedby={describe('email-hint', err('email') && 'email-error')}
              maxlength="254"
            />
            {#if err('email')}<p class="field-error" id="email-error">{err('email')}</p>{/if}
          </div>
        </fieldset>

        <!-- Honeypot (contract §6): people never see or reach this; bots fill it in. -->
        <div class="hp" aria-hidden="true">
          <label for="website">Website (leave this empty)</label>
          <input id="website" name="website" type="text" tabindex="-1" autocomplete="off" value={form.website} oninput={setText('website')} />
        </div>

        <section class="privacy" aria-labelledby="privacy-title">
          <h2 id="privacy-title">What we keep</h2>
          <ul>
            <li>What you type in this form.</li>
            <li>
              A scrambled (hashed) form of your IP address, with a key that changes every day, used only to stop spam and
              abuse. Never the address itself.
            </li>
            <li>Your email, only if you give it, until the submission is closed.</li>
          </ul>
          <p>No account, no cookies for this form, no tracking and no ads.</p>
        </section>

        <p class="actions">
          <button type="submit" class="button primary" disabled={form.phase === 'submitting'}>
            {form.phase === 'submitting' ? 'Sending…' : 'Send it'}
          </button>
          {#if form.target}<a href="/wood/{form.target.slug}">Cancel</a>{/if}
        </p>
        <p class="visually-hidden" aria-live="polite">{form.phase === 'submitting' ? 'Sending your submission.' : ''}</p>
      {/if}
    </form>
  {/if}
</div>

<style>
  .submit {
    max-width: 46rem;
    padding-top: 1.5rem;
  }

  .intro {
    display: flex;
    gap: 1.25rem;
    align-items: center;
    margin-bottom: 1rem;
  }

  .intro :global(.intro-art) {
    flex: none;
    width: 120px;
  }

  .lede {
    color: var(--muted);
    margin: 0;
  }

  fieldset {
    border: 0;
    margin: 0;
    padding: 0;
    min-width: 0;
  }

  legend {
    font-family: var(--font-serif);
    font-size: 1.3rem;
    font-weight: 700;
    padding: 0;
    margin-bottom: 0.75rem;
  }

  .group {
    margin-top: 2rem;
    padding-top: 1.25rem;
    border-top: 1px solid var(--rule);
  }

  .kinds {
    margin-top: 1.5rem;
  }

  .kind-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
    gap: 0.6rem;
  }

  .kind {
    display: flex;
    gap: 0.65rem;
    align-items: flex-start;
    padding: 0.75rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    cursor: pointer;
  }

  .kind:hover {
    background: var(--surface-2);
  }

  .kind.selected {
    border-color: var(--blue);
    box-shadow: inset 0 0 0 1px var(--blue);
  }

  .kind:has(input:focus-visible) {
    outline: 3px solid var(--focus);
    outline-offset: 2px;
  }

  .kind input {
    margin-top: 0.3rem;
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--blue);
    flex: none;
  }

  .kind-text {
    display: flex;
    flex-direction: column;
  }

  .kind-label {
    font-weight: 650;
  }

  .kind-hint {
    font-size: 0.9rem;
    color: var(--muted);
  }

  .medical {
    margin-top: 1.5rem;
    padding: 1rem 1.15rem;
    border: 2px solid var(--accent);
    border-left-width: 8px;
    border-radius: var(--radius-sm);
    background: var(--surface);
  }

  .medical h2 {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 1.3rem;
  }

  .medical p:last-child {
    margin-bottom: 0;
  }

  .field {
    margin-bottom: 1rem;
  }

  .field label {
    display: block;
    font-weight: 650;
    margin-bottom: 0.3rem;
  }

  .row.two {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 0 1rem;
  }

  input[type='text'],
  input[type='email'],
  input[type='url'],
  select,
  textarea {
    width: 100%;
    font: inherit;
    font-size: 1rem;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 0.55rem 0.7rem;
    min-height: 44px;
  }

  textarea {
    resize: vertical;
    line-height: 1.5;
  }

  [aria-invalid='true'] {
    border-color: var(--accent);
    border-width: 2px;
  }

  .req,
  .opt {
    font-weight: 400;
    font-size: 0.9rem;
    color: var(--muted);
    font-family: var(--font-sans);
  }

  .hint {
    font-size: 0.92rem;
    color: var(--muted);
    margin: 0 0 0.4rem;
  }

  .own-words {
    font-size: 0.92rem;
    margin: 0 0 0.5rem;
    padding: 0.4rem 0.7rem;
    border-left: 3px solid var(--wood);
    background: var(--surface-2);
  }

  .counter {
    font-size: 0.88rem;
    color: var(--muted);
    margin: 0.25rem 0 0;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .counter.over {
    color: var(--accent);
    font-weight: 650;
  }

  .field-error {
    color: var(--accent);
    font-weight: 600;
    font-size: 0.95rem;
    margin: 0.3rem 0 0.5rem;
  }

  .field-error::before {
    content: '✕ ';
  }

  .field-warn {
    font-size: 0.93rem;
    margin: 0.3rem 0 0.5rem;
    padding: 0.4rem 0.7rem;
    border-left: 3px solid var(--wood);
    background: var(--ev3-bg);
    color: var(--ev3-fg);
  }

  .chosen {
    display: flex;
    gap: 1rem;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    padding: 0.6rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .chosen p {
    margin: 0;
    display: flex;
    flex-direction: column;
  }

  .chosen-name {
    font-weight: 700;
    font-size: 1.1rem;
  }

  .cites {
    list-style: none;
    margin: 0 0 1rem;
    padding: 0;
  }

  .cite {
    padding: 0.9rem 0.9rem 0.4rem;
    margin-bottom: 0.75rem;
    border: 1px dashed var(--border);
    border-radius: var(--radius);
  }

  .cite-grid {
    display: grid;
    grid-template-columns: 11rem 1fr;
    gap: 0 0.75rem;
  }

  @media (max-width: 34rem) {
    .cite-grid {
      grid-template-columns: 1fr;
    }
  }

  .cite .field {
    margin-bottom: 0.6rem;
  }

  .linkish {
    font: inherit;
    background: none;
    border: 0;
    padding: 0.3rem 0;
    color: var(--blue);
    text-decoration: underline;
    text-underline-offset: 0.15em;
    cursor: pointer;
    min-height: 32px;
  }

  .remove {
    margin-bottom: 0.4rem;
    font-size: 0.93rem;
  }

  .hp {
    position: absolute;
    left: -10000px;
    top: auto;
    width: 1px;
    height: 1px;
    overflow: hidden;
  }

  .privacy {
    margin-top: 2rem;
    padding: 0.9rem 1rem;
    background: var(--surface-2);
    border-radius: var(--radius);
    font-size: 0.93rem;
  }

  .privacy h2 {
    font-size: 1.05rem;
    margin: 0 0 0.4rem;
  }

  .privacy ul {
    margin: 0 0 0.5rem;
    padding-left: 1.2rem;
  }

  .privacy p {
    margin: 0;
  }

  .actions {
    display: flex;
    gap: 1.25rem;
    align-items: center;
    margin-top: 1.5rem;
  }

  .actions .button {
    min-width: 9rem;
    justify-content: center;
  }

  .actions .button[disabled] {
    opacity: 0.7;
    cursor: progress;
  }

  .error-summary,
  .banner {
    margin: 1.25rem 0;
    padding: 0.9rem 1rem;
    border: 2px solid var(--accent);
    border-radius: var(--radius-sm);
    background: var(--surface);
  }

  .error-summary h2 {
    font-size: 1.15rem;
    margin: 0 0 0.4rem;
    color: var(--accent);
  }

  .error-summary ul {
    margin: 0;
    padding-left: 1.2rem;
  }

  .banner p {
    margin: 0;
  }

  .done {
    margin-top: 1rem;
  }

  .done h2 {
    margin-top: 0;
  }

  .ref {
    font-family: var(--font-mono);
    font-size: 1.15rem;
  }

  .save {
    padding: 1rem;
    border: 2px solid var(--blue);
    border-radius: var(--radius);
    background: var(--blue-soft);
    margin: 1rem 0;
  }

  .status-link {
    font-family: var(--font-mono);
    font-size: 0.95rem;
    overflow-wrap: anywhere;
    padding: 0.5rem 0.7rem;
    background: var(--surface);
    border-radius: var(--radius-sm);
  }

  .warn {
    padding: 0.75rem 1rem;
    border-left: 4px solid var(--wood);
    background: var(--ev3-bg);
    color: var(--ev3-fg);
    margin: 1rem 0;
  }

  .warn p {
    margin: 0 0 0.25rem;
  }

  .warn ul {
    margin: 0;
    padding-left: 1.2rem;
  }

  .next {
    display: flex;
    gap: 1.25rem;
    align-items: center;
    flex-wrap: wrap;
  }

  @media (max-width: 34rem) {
    .intro :global(.intro-art) {
      width: 80px;
    }
  }
</style>
