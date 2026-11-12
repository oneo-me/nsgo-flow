import { defineConfig } from 'astro/config';

// The site is deployed at /nsgo-flow/ of oneo-me.github.io on GitHub Pages
// by .github/workflows/website.yml. Every page is static; English is at the
// base and Chinese under zh/.
export default defineConfig({
  site: 'https://oneo-me.github.io',
  base: '/nsgo-flow/',
  trailingSlash: 'always',
  devToolbar: { enabled: false },
});
