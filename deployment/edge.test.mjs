import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import vm from 'node:vm'
import { buildEdge } from './build-edge.mjs'

const workspace = await mkdtemp(path.join(tmpdir(), 'cjv-edge-test-'))
const siteDir = path.join(workspace, 'site')
const edgeDir = path.join(workspace, 'edgeone', '.edgeone')
const cfDir = path.join(workspace, 'cloudflare')
const html = '<!doctype html><html lang="zh-CN"><body><script src="/assets/app.js"></script></body></html>'
let edgeHandler
let worker

before(async () => {
  await mkdir(path.join(siteDir, 'assets'), { recursive: true })
  await writeFile(path.join(siteDir, 'index.html'), html)
  await writeFile(path.join(siteDir, 'assets/app.js'), 'console.log("app")')
  await writeFile(path.join(siteDir, '404.html'), 'Missing')
  await buildEdge('edgeone', siteDir, edgeDir)
  await buildEdge('cloudflare', '-', cfDir)
  const bundle = await readFile(path.join(edgeDir, 'edge-functions/index.js'), 'utf8')
  assert.ok(bundle.startsWith("import 'hooks';"))
  vm.runInNewContext(bundle.replace("import 'hooks';", ''), {
    Response,
    URL,
    addEventListener(event, handler) {
      assert.equal(event, 'fetch')
      edgeHandler = handler
    },
  })
  worker = (await import(pathToFileURL(path.join(cfDir, 'workers/default/bundle/worker.mjs')))).default
})

after(async () => {
  assert.equal(path.dirname(workspace), path.resolve(tmpdir()))
  await rm(workspace, { recursive: true })
})

for (const provider of ['edgeone', 'cloudflare']) {
  for (const country of ['CN', 'US', 'HK', undefined]) {
    test(`${provider}: first HTML response defaults mirror for ${country}`, async t => {
      const request = new Request('https://example.com/')
      let response
      if (provider === 'edgeone') {
        request.eo = { geo: { countryCodeAlpha2: country } }
        edgeHandler({ request, respondWith(value) { response = value } })
      } else {
        request.cf = { country }
        t.mock.method(globalThis, 'fetch', async () => new Response(html, {
          headers: { 'Content-Type': 'text/html', 'Cache-Control': 'public, max-age=600', ETag: 'old' },
        }))
        response = await worker.fetch(request)
      }
      assert.equal(response.status, 200)
      assert.match(await response.text(), new RegExp(`<html data-default-mirror="${country === 'CN'}" lang="zh-CN">`))
      assert.equal(response.headers.get('cache-control'), 'private, no-store')
      assert.equal(response.headers.get('etag'), null)
    })
  }
}

test('EdgeOne routes the homepage to the function before static files, preserving 404s and assets', async () => {
  const routes = JSON.parse(await readFile(path.join(edgeDir, 'routes.json'), 'utf8')).routes
  assert.equal(routes[0]['server-name'], 'edge')
  for (const pathname of ['/', '/index.html']) assert.match(pathname, new RegExp(routes[0].src))
  for (const pathname of ['/assets/app.js', '/book/', '/missing']) assert.doesNotMatch(pathname, new RegExp(routes[0].src))
  assert.equal(routes[1].handle, 'filesystem')
  assert.equal(routes[2].status, 404)
  assert.equal(await readFile(path.join(edgeDir, 'assets/assets/app.js'), 'utf8'), 'console.log("app")')
  assert.equal(await readFile(path.join(edgeDir, 'assets/404.html'), 'utf8'), 'Missing')
})

test('EdgeOne rejects upload roots that the CLI would interpret as plain static files', async () => {
  await assert.rejects(buildEdge('edgeone', siteDir, path.join(workspace, 'plain-output')), /must be named \.edgeone/)
})

test('EdgeOne HEAD returns the same cache policy without a body', async () => {
  let response
  edgeHandler({ request: new Request('https://example.com/', { method: 'HEAD' }), respondWith(value) { response = value } })
  assert.equal(response.status, 200)
  assert.equal(await response.text(), '')
  assert.equal(response.headers.get('cache-control'), 'private, no-store')
})

test('EdgeOne rejects unsupported methods', () => {
  let response
  edgeHandler({ request: new Request('https://example.com/', { method: 'POST' }), respondWith(value) { response = value } })
  assert.equal(response.status, 405)
  assert.equal(response.headers.get('allow'), 'GET, HEAD')
})

for (const pathname of ['/assets/app.js', '/book/user-guide/zh-CN/', '/missing']) {
  test(`EdgeOne forwards ${pathname} through the platform origin hook`, async () => {
    let response
    const request = new Request(`https://example.com${pathname}`)
    const original = new Response('origin content', { status: pathname === '/missing' ? 404 : 200 })
    edgeHandler({ request, respondWith(value) { response = value } }, {
      fetch(forwarded) {
        assert.equal(forwarded, request)
        return original
      },
    })
    assert.equal(await response, original)
    assert.equal(await (await response).text(), 'origin content')
  })
}

test('Cloudflare strips conditional headers and stale body metadata, preserves security headers, and uses fresh origin HTML', async t => {
  const fresh = html.replace('<body>', '<body>latest origin')
  t.mock.method(globalThis, 'fetch', async request => {
    assert.equal(request.headers.get('if-none-match'), null)
    assert.equal(request.headers.get('if-modified-since'), null)
    return new Response(fresh, { headers: {
      'Content-Type': 'text/html',
      'Content-Length': '1',
      'Content-Encoding': 'gzip',
      'Last-Modified': 'Wed, 21 Oct 2026 00:00:00 GMT',
      'X-Content-Type-Options': 'nosniff',
    } })
  })
  const request = new Request('https://example.com/', { headers: { 'If-None-Match': 'old', 'If-Modified-Since': 'old' } })
  request.cf = { country: 'CN' }
  const response = await worker.fetch(request)
  assert.match(await response.text(), /latest origin/)
  for (const key of ['content-length', 'content-encoding', 'last-modified']) assert.equal(response.headers.get(key), null)
  assert.equal(response.headers.get('x-content-type-options'), 'nosniff')
})

test('Cloudflare preserves origin errors instead of turning them into successful homepages', async t => {
  const origin = new Response('Missing', { status: 404, headers: { 'Content-Type': 'text/html' } })
  t.mock.method(globalThis, 'fetch', async () => origin)
  assert.equal(await worker.fetch(new Request('https://example.com/')), origin)
})

test('Cloudflare HEAD does not send a response body', async t => {
  t.mock.method(globalThis, 'fetch', async () => new Response(null, { headers: { 'Content-Type': 'text/html' } }))
  const response = await worker.fetch(new Request('https://example.com/', { method: 'HEAD' }))
  assert.equal(await response.text(), '')
  assert.equal(response.headers.get('cache-control'), 'private, no-store')
})
