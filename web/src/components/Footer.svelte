<script lang="ts">
  import { app } from '../lib/nav.svelte';
</script>

<footer class="site-footer">
  <!-- Art slot: images/art/border-forest.png (site-art.md §7), decorative; nothing shows if missing. -->
  <!-- style: directive, not a style="" attribute: Svelte sets it through CSSOM, which CSP style-src 'self' allows. -->
  <div class="forest" aria-hidden="true" style:background-image="url('/images/art/border-forest.png')"></div>
  <div class="double-rule" aria-hidden="true"></div>
  <div class="wrap cols">
    <section aria-labelledby="footer-medical">
      <h2 id="footer-medical">Not medical advice</h2>
      <p>
        This site is information only. If you have symptoms such as a cough, wheeze, a stuffy nose that clears on
        days away from the shop, nosebleeds or a rash, see a doctor and tell them which woods you work with.
      </p>
    </section>
    <section aria-labelledby="footer-data">
      <h2 id="footer-data">About the data</h2>
      <p>
        Every hazard links to its sources and carries an evidence level. Absence of reports is not evidence that
        a wood is safe. Data license: CC BY 4.0 (pending confirmation).
      </p>
    </section>
    <nav aria-label="Footer">
      <h2>More</h2>
      <ul>
        <li><a href="/methodology">Methodology and evidence levels</a></li>
        <li><a href="/sources">All sources</a></li>
        <li><a href="/symptoms">Symptoms to watch for</a></li>
        <li><a href="/limits">Exposure limits</a></li>
        <li><a href="/glossary">Glossary</a></li>
      </ul>
    </nav>
  </div>
  <p class="wrap small">
    {app.site.name}. No tracking, no ads, no affiliate links.
    <span class="suggest">Know a wood we’re missing? <a href="/submit?kind=new_species">Suggest a species</a>.</span>
  </p>
</footer>

<style>
  .site-footer {
    background: var(--surface-2);
    color: var(--text);
    margin-top: 4rem;
  }

  .forest {
    height: 64px;
    margin-top: -64px;
    background-repeat: repeat-x;
    background-size: auto 64px;
    background-position: bottom left;
  }

  .cols {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
    gap: 1.5rem 2.5rem;
    padding-top: 1.75rem;
  }

  h2 {
    font-size: 1.05rem;
    margin: 0 0 0.5rem;
  }

  p {
    font-size: 0.95rem;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    margin: 0.15rem 0;
  }

  li a {
    display: inline-block;
    padding: 0.3rem 0;
  }

  .small {
    font-size: 0.85rem;
    color: var(--muted);
    padding-bottom: 1.5rem;
    margin: 0 auto;
  }

  .suggest {
    margin-left: 0.5rem;
  }

  .small a {
    color: inherit;
  }

  @media print {
    .site-footer {
      background: none;
      margin-top: 4pt;
    }

    .double-rule,
    .forest,
    nav,
    .small,
    section[aria-labelledby='footer-data'] {
      display: none;
    }

    .cols {
      padding-top: 0;
      display: block;
    }

    h2 {
      display: inline;
      font-size: 8pt;
    }

    p {
      display: inline;
      font-size: 8pt;
    }
  }
</style>
