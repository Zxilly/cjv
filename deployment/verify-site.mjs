import assert from 'node:assert/strict'
import { setTimeout } from 'node:timers/promises'

async function get(url, method = 'GET') {
  return fetch(url, { method, cache: 'no-store', signal: AbortSignal.timeout(15000) })
}

async function verify(site) {
  const response = await get(site)
  assert.equal(response.status, 200, `${site}: homepage status`)
  const html = await response.text()
  assert.match(html, /<html\b[^>]*data-default-mirror="(?:true|false)"/, `${site}: missing edge injection`)
  assert.match(response.headers.get('cache-control') || '', /no-store/, `${site}: homepage cache policy`)
  const asset = html.match(/src="(\/assets\/[^"\s]+\.js)"/)?.[1]
  assert.ok(asset, `${site}: missing application bundle`)
  const bundleResponse = await get(new URL(asset, site))
  assert.equal(bundleResponse.status, 200, `${site}: application bundle status`)
  const bundle = await bundleResponse.text()
  assert.ok(bundle.includes('dataset.defaultMirror'), `${site}: application does not read injected default`)
  assert.ok(!bundle.includes('get.geojs.io'), `${site}: obsolete GeoJS request`)
  const llmsResponse = await get(new URL('llms.txt', site))
  assert.equal(llmsResponse.status, 200, `${site}: llms.txt status`)
  assert.match(llmsResponse.headers.get('content-type') || '', /text\/plain/, `${site}: llms.txt content type`)
  assert.match(await llmsResponse.text(), /^# cjv - Cangjie Version Manager\r?\n/, `${site}: llms.txt content`)
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

const sites = process.argv.slice(2)
if (!sites.length) throw new Error('Usage: node deployment/verify-site.mjs <site URL> [...]')
await Promise.all(sites.map(verifyWithRetry))
