import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { createServer } from 'vite'

const server = await createServer({ server: { middlewareMode: true, hmr: false, watch: null }, appType: 'custom' })
const escape = value => value.replaceAll('&', '&amp;').replaceAll('"', '&quot;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
try {
  const { render, PAGE_META, SITE_URL } = await server.ssrLoadModule('/src/entry-server.tsx')
  const template = await readFile('dist/index.html', 'utf8')
  for (const [lang, meta] of Object.entries(PAGE_META)) {
    const url = SITE_URL + meta.path
    const head = `
  <link rel="canonical" href="${url}">
  <link rel="alternate" hreflang="zh-CN" href="${SITE_URL}/">
  <link rel="alternate" hreflang="en" href="${SITE_URL}/en/">
  <link rel="alternate" hreflang="x-default" href="${SITE_URL}/">
  <meta property="og:type" content="website">
  <meta property="og:site_name" content="cjv">
  <meta property="og:title" content="${escape(meta.title)}">
  <meta property="og:description" content="${escape(meta.description)}">
  <meta property="og:url" content="${url}">
  <meta property="og:locale" content="${meta.locale}">
  <meta name="twitter:card" content="summary">
  <meta name="twitter:title" content="${escape(meta.title)}">
  <meta name="twitter:description" content="${escape(meta.description)}">
  <script type="application/ld+json">${JSON.stringify({
    '@context': 'https://schema.org', '@type': 'SoftwareApplication', name: 'cjv',
    url: SITE_URL, description: meta.description, applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Windows, macOS, Linux, HarmonyOS',
    license: 'https://www.apache.org/licenses/LICENSE-2.0',
    sameAs: ['https://github.com/Zxilly/cjv', 'https://gitcode.com/Zxilly/cjv'],
  }).replaceAll('<', '\\u003c')}</script>`
    const html = template
      .replace('<html lang="zh-CN">', `<html lang="${meta.language}" data-prerendered="true" data-page-lang="${lang}">`)
      .replace(/<title>.*?<\/title>/, `<title>${escape(meta.title)}</title>`)
      .replace(/<meta name="description"[^>]*>/, `<meta name="description" content="${escape(meta.description)}">`)
      .replace('</head>', head + '\n</head>')
      .replace('<div id="app" class="w-full"></div>', () => `<div id="app" class="w-full">${render(lang)}</div>`)
    const directory = lang === 'en' ? 'dist/en' : 'dist'
    await mkdir(directory, { recursive: true })
    await writeFile(`${directory}/index.html`, html)
  }
  await writeFile('dist/robots.txt', `User-agent: *\nAllow: /\n\nSitemap: ${SITE_URL}/sitemap.xml\n`)
  await writeFile('dist/sitemap.xml', `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${Object.values(PAGE_META).map(meta => `<url><loc>${SITE_URL + meta.path}</loc></url>`).join('')}</urlset>\n`)
  console.log('Prerendered Chinese and English homepages.')
} finally {
  await server.close()
}
