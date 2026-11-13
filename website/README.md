# Scenra website

The official site at <https://oneo-me.github.io/nsgo-scenra/>: static Astro pages
in English and Chinese, deployed to GitHub Pages by the repository's Website
workflow.

```sh
bun install
bun run dev          # serve the site at http://localhost:4321/nsgo-scenra/
bun run build        # typecheck, then build dist/
bun run preview      # serve dist/
```

## Pages

Paths are below <https://oneo-me.github.io/nsgo-scenra/>.

| Page | Path |
| --- | --- |
| Home | `/` and `/zh/` |
| Privacy | `/privacy/` and `/zh/privacy/` |
| Not found | `/404.html` |

`src/copy.ts` holds every word in both languages; change English and Chinese
together. Chinese headings, titles and kickers omit the trailing `。`.
`src/pages/[...lang]/` builds each page for both languages, `src/site.ts` lists
the languages and outside links, and `src/layouts/Layout.astro` gives every page
its head, header, footer and scripts. A first visit from a Chinese browser
opens the Chinese page unless another language was chosen.

`public/` holds the icon, the app screenshot, the Open Graph image, the
favicon, `robots.txt` and `sitemap.xml`. The icon and screenshot are copies of
the app's `assets/`; render `scripts/og.html` at 1200×630 with Chrome to
regenerate the Open Graph image from them. A change to the site's address
belongs in `astro.config.mjs`, the Umami script's `data-domains` in
`Layout.astro`, the sitemap and the Open Graph image together.

## Statistics

`Layout.astro` loads Umami with Scenra's website ID, tagged `website`, so
visits are counted beside the app's own use. `data-domains` keeps local servers
and other hosts out. The privacy page describes what is counted and what is never sent.

## Deployment

The [Website workflow](../.github/workflows/website.yml) builds `website/dist`
on pushes to `main` and publishes it to GitHub Pages from GitHub Actions. No
custom domain is configured: the site is the repository's project page,
`https://oneo-me.github.io/nsgo-scenra/`. `astro.config.mjs` fixes the site and
base for that path; another address needs both changed.
