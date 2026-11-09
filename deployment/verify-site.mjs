import assert from 'node:assert/strict'
import { setTimeout } from 'node:timers/promises'

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
      assert.ok(page.includes(`<link rel="canonical" href="${canonical}">`), `${site}${pathname}: canonical`)
      assert.match(page, /<h1\b/, `${site}${pathname}: missing static heading`)
      assert.ok(page.includes('toolchain-file.html'), `${site}${pathname}: missing static document links`)
      if (new URL(site).hostname === 'cjv.zxilly.dev') assert.ok(page.includes('https://cjv.zxilly.dev/install.sh'), `${site}${pathname}: installer domain`)
    }
    const sitemap = await get(new URL('/sitemap.xml', site))
    assert.equal(sitemap.status, 200, `${site}: sitemap status`)
    assert.ok((await sitemap.text()).includes('/book/user-guide/zh-CN/installation/'), `${site}: missing documentation sitemap entry`)
    const robots = await get(new URL('/robots.txt', site))
    assert.equal(robots.status, 200, `${site}: robots status`)
    assert.ok((await robots.text()).includes('Sitemap: https://cjv.zxilly.com/sitemap.xml'), `${site}: sitemap discovery`)
  }
  assert.match(html, /<html\b[^>]*data-default-mirror="(?:true|false)"/, `${site}: missing edge injection`)
  assert.match(response.headers.get('cache-control') || '', /no-store/, `${site}: homepage cache policy`)
  const asset = html.match(/src="(\/assets\/[^"\s]+\.js)"/)?.[1]
  assert.ok(asset, `${site}: missing application bundle`)
  const bundleResponse = await get(new URL(asset, site))
  assert.equal(bundleResponse.status, 200, `${site}: application bundle status`)
  const bundle = await bundleResponse.text()
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
