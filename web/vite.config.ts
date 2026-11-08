import { defineConfig, type PluginOption, type UserConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The SPA is built to `dist/` and embedded into the Go binary (web/embed.go).
// Go replaces the three <!--ssr-…--> markers in dist/index.html on every page
// response (docs/contract.md §1), so the build must leave them intact.
//
// Dev: the Vite server proxies the API and image paths to the Go server on
// 127.0.0.1:$PORT (default 8080). With VITE_FIXTURES=1 the dev server instead
// answers /api/v1/* and /images/* itself from ../data and ../content/pages via
// fixtures/server.ts. That module is only imported here, in Node, during
// `vite serve`; it is never part of the client bundle.
const apiTarget = `http://127.0.0.1:${process.env.PORT || '8080'}`;

export default defineConfig(async ({ command }): Promise<UserConfig> => {
  const fixtures = command === 'serve' && process.env.VITE_FIXTURES === '1';
  const plugins: PluginOption[] = [svelte()];
  if (fixtures) {
    const { fixturesPlugin } = await import('./fixtures/server.ts');
    plugins.push(fixturesPlugin());
  }
  return {
    plugins,
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      // Keep the shell's HTML comments (the SSR markers) and avoid inlining
      // tiny assets into the HTML that Go rewrites.
      assetsInlineLimit: 0,
    },
    server: {
      port: 5173,
      strictPort: true,
      proxy: fixtures
        ? {}
        : {
            '/api': apiTarget,
            '/images': apiTarget,
            '/sitemap.xml': apiTarget,
          },
    },
  };
});
