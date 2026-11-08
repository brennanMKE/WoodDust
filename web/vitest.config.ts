import { defineConfig } from 'vitest/config';

// Unit tests cover pure TypeScript modules (router, combobox, labels), so they
// run in Node without the Svelte plugin or a DOM.
export default defineConfig({
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'node',
  },
});
