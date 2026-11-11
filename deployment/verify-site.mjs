import assert from 'node:assert/strict'
import { setTimeout } from 'node:timers/promises'
import posts from '../web/src/content/blog/posts.json' with { type: 'json' }

const verifySEO = process.argv.includes('--seo')

async function get(url, method = 'GET') {
  return fetch(url, { method, cache: 'no-store', signal: AbortSignal.timeout(15000) })
}

async function verify(site) {
  const response = await get(site)
  assert.equal(response.status, 200, `${site}: homepage status`)
  const html = await response.text()
  if (verifySEO) {
    for (const [pathname, language, canonical] of [['/', 'zh-CN', 'https://cjv.zxilly.com/'], ['/en/', 'en', 'https://cjv.zxilly.com/en/']]) {
      const page = pathname === '/' ? html : await (await get(new URL(pathname, site))).text()
      assert.match(page, new RegExp(`<html[^>]*lang="${language}"`), `${site}${pathname}: language`)
      assert.ok(page.includes('data-prerendered="true"'), `${site}${pathname}: missing prerender`)
      assert.ok(page.includes(`<link rel="canonical" href="${canonical}"`), `${site}${pathname}: canonical`)
      assert.match(page, /<h1\b/, `${site}${pathname}: missing static heading`)
      assert.ok(page.includes(`href="https://cjv.zxilly.com/book/user-guide/${language === 'en' ? 'en' : 'zh-CN'}/"`), `${site}${pathname}: missing static document link`)
      if (new URL(site).hostname === 'cjv.zxilly.dev') assert.ok(page.includes('https://cjv.zxilly.dev/install.sh'), `${site}${pathname}: installer domain`)
    }
    const sitemap = await get(new URL('/sitemap.xml', site))
    assert.equal(sitemap.status, 200, `${site}: sitemap status`)
    const sitemapXML = await sitemap.text()
    assert.ok(sitemapXML.includes('/book/user-guide/zh-CN/installation/'), `${site}: missing documentation sitemap entry`)
    for (const prefix of ['', '/en']) {
      for (const slug of ['', ...posts.map(post => `${post.slug}/`)]) {
        const path = `${prefix}/blog/${slug}`
        const pageResponse = await get(new URL(path, site))
        assert.equal(pageResponse.status, 200, `${site}${path}: Blog status`)
        const page = await pageResponse.text()
        assert.ok(page.includes(`<link rel="canonical" href="https://cjv.zxilly.com${path}"`), `${site}${path}: Blog canonical`)
        assert.match(page, /<h1\b/, `${site}${path}: static Blog content`)
        assert.ok(sitemapXML.includes(`https://cjv.zxilly.com${path}</loc>`), `${site}${path}: Blog sitemap entry`)
        if (slug) {
          assert.ok(page.includes('"@type":"BlogPosting"'), `${site}${path}: article schema`)
          const data = await get(new URL(`${path}_.data`, site))
          assert.equal(data.status, 200, `${site}${path}: navigation data`)
          assert.ok(!(await data.text()).trimStart().startsWith('<'), `${site}${path}: navigation data must not be HTML`)
        }
      }
    }
    const robots = await get(new URL('/robots.txt', site))
    assert.equal(robots.status, 200, `${site}: robots status`)
    assert.ok((await robots.text()).includes('Sitemap: https://cjv.zxilly.com/sitemap.xml'), `${site}: sitemap discovery`)
    const imageURL = 'https://cjv.zxilly.com/og/cjv.png'
    const image = await get(new URL('/og/cjv.png', site))
    assert.equal(image.status, 200, `${site}: sharing image status`)
    assert.match(image.headers.get('content-type') || '', /image\/png/, `${site}: sharing image type`)
    const imageBytes = Buffer.from(await image.arrayBuffer())
    assert.equal(imageBytes.subarray(0, 8).toString('hex'), '89504e470d0a1a0a', `${site}: PNG signature`)
    assert.equal(imageBytes.readUInt32BE(16), 1734, `${site}: sharing image width`)
    assert.equal(imageBytes.readUInt32BE(20), 907, `${site}: sharing image height`)
    for (const book of ['user-guide', 'dev-guide']) {
      for (const language of ['zh-CN', 'en']) {
        const routes = ['', ...(book === 'user-guide' ? ['installation/'] : [])]
        for (const route of routes) {
          const path = `/book/${book}/${language}/${route}`
          const docResponse = await get(new URL(path, site))
          assert.equal(docResponse.status, 200, `${site}${path}: documentation status`)
          const doc = await docResponse.text()
          const head = doc.match(/<head\b[^>]*>([\s\S]*?)<\/head>/i)?.[1] || ''
          assert.ok(head.includes(`<link rel="canonical" href="https://cjv.zxilly.com${path}"`), `${site}${path}: documentation canonical`)
          for (const alternate of ['zh-CN', 'en']) {
            assert.ok(head.includes(`hreflang="${alternate}" href="https://cjv.zxilly.com/book/${book}/${alternate}/${route}"`), `${site}${path}: language alternate ${alternate}`)
          }
          assert.ok(head.includes(`<meta property="og:image" content="${imageURL}"`), `${site}${path}: documentation sharing image`)
          assert.ok(head.includes('<meta name="twitter:card" content="summary_large_image"'), `${site}${path}: large sharing card`)
          assert.equal((head.match(/name="description"/g) || []).length, 1, `${site}${path}: one description`)
          assert.equal((head.match(/trail\.learningman\.top\/script\.js/g) || []).length, 1, `${site}${path}: analytics preserved`)
          assert.ok(!head.includes('noindex'), `${site}${path}: documentation remains indexable`)
          assert.ok(sitemapXML.includes(`https://cjv.zxilly.com${path}</loc>`), `${site}${path}: documentation sitemap entry`)
        }
        const print = await get(new URL(`/book/${book}/${language}/print.html`, site))
        assert.equal(print.status, 200, `${site}: print page status`)
        assert.ok((await print.text()).includes('<meta name="robots" content="noindex,follow"'), `${site}: print page excluded from indexing`)
      }
    }
  }
  assert.match(html, /<html\b[^>]*data-default-mirror="(?:true|false)"/, `${site}: missing edge injection`)
  assert.match(response.headers.get('cache-control') || '', /no-store/, `${site}: homepage cache policy`)
  // Framework builds load split modules via modulepreload and an inline entry.
  const assets = [...new Set([...html.matchAll(/(?:src|href)="(\/assets\/[^"\s]+\.js)"/g)].map(match => match[1]))]
  assert.ok(assets.length, `${site}: missing application bundles`)
  const bundle = (await Promise.all(assets.map(async asset => {
    const response = await get(new URL(asset, site))
    assert.equal(response.status, 200, `${site}: application bundle ${asset} status`)
    return response.text()
  }))).join('\n')
  assert.ok(bundle.includes('dataset.defaultMirror'), `${site}: application does not read injected default`)
  assert.ok(!bundle.includes('get.geojs.io'), `${site}: obsolete GeoJS request`)
  for (const route of ['install.sh', 'book/user-guide/zh-CN/', 'dl/mirror/linux_amd64/cjv-init']) {
    const staticResponse = await get(new URL(route, site), 'HEAD')
    assert.equal(staticResponse.status, 200, `${site}: static route ${route}`)
  }
  const missing = await get(new URL('__cjv_missing_file_check__', site), 'HEAD')
  assert.equal(missing.status, 404, `${site}: missing files must return 404`)
  console.log(`${site}: edge injection, application bundle, static routes, and 404 verified`)
}

async function verifyWithRetry(site) {
  for (let attempt = 1; ; attempt++) {
    try {
      await verify(site)
      return
    } catch (error) {
      if (attempt >= 6) throw error
      console.log(`${site}: waiting for deployment propagation (${attempt}/6): ${error.message}`)
      await setTimeout(10000)
    }
  }
}

const sites = process.argv.slice(2).filter(value => value !== '--seo')
if (!sites.length) throw new Error('Usage: node deployment/verify-site.mjs <site URL> [...]')
await Promise.all(sites.map(verifyWithRetry))
