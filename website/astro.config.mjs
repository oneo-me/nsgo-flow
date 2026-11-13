import { defineConfig } from 'astro/config';

// The site is deployed at /nsgo-scenra/ of oneo-me.github.io on GitHub Pages
// by .github/workflows/website.yml. Every page is static; English is at the
// base and Chinese under zh/.
export default defineConfig({
  site: 'https://oneo-me.github.io',
  base: '/nsgo-scenra/',
  trailingSlash: 'always',
  devToolbar: { enabled: false },
});
