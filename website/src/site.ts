import { copy, type Copy } from './copy';

// The site's languages, addresses and outside links. English lives at the
// base and Chinese under zh/.

export type Lang = 'en' | 'zh';
export const langs: Lang[] = ['en', 'zh'];

/** The path of `path` in `lang`, for the site's root. */
export function localePath(lang: Lang, path = ''): string {
  return `${import.meta.env.BASE_URL}${lang === 'en' ? '' : `${lang}/`}${path}`;
}

export function otherLang(lang: Lang): Lang {
  return lang === 'en' ? 'zh' : 'en';
}

export function langOf(param: string | undefined): Lang {
  return param === 'zh' ? 'zh' : 'en';
}

/** getStaticPaths for a page under src/pages/[...lang]/. */
export function langPaths() {
  return langs.map((lang) => ({ params: { lang: lang === 'en' ? undefined : lang } }));
}

/** langAlternate is the hreflang of a language. */
export const langAlternate = (lang: Lang) => (lang === 'zh' ? 'zh-CN' : 'en');

const repo = 'https://github.com/oneo-me/nsgo-scenra';

export const links = {
  repo,
  issues: `${repo}/issues`,
  readme: `${repo}#readme`,
  changelog: `${repo}/blob/main/CHANGELOG.md`,
  license: `${repo}/blob/main/LICENSE`,
  examples: `${repo}/tree/main/examples`,
  nsgo: 'https://nsgo.dev',
  blog: 'https://oneo.me',
  x: 'https://x.com/oneo666',
  sponsors: 'https://github.com/sponsors/oneo-me',
};

/** pageCopy is a page's words in a language. */
export const pageCopy = (lang: Lang): Copy => copy[lang];
