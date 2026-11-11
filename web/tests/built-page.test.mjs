import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { chromium, firefox, webkit } from 'playwright'
import { preview } from 'vite'
import { homeResponse } from '../../deployment/home-response.mjs'

let server
let browser
let origin
const browserName = process.env.PLAYWRIGHT_BROWSER || 'chromium'
const blogPosts = JSON.parse(await readFile(new URL('../src/content/blog/posts.json', import.meta.url), 'utf8'))

before(async () => {
  server = await preview({
    root: fileURLToPath(new URL('..', import.meta.url)),
    configFile: false,
    appType: 'mpa',
    build: { outDir: 'dist/client' },
    logLevel: 'error',
    preview: { host: '127.0.0.1', port: 0 },
  })
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await ({ chromium, firefox, webkit })[browserName].launch()
})

after(async () => {
  await browser?.close()
  if (server) await new Promise((resolve, reject) => server.httpServer.close(error => error ? reject(error) : resolve()))
})

test('every translated Blog article is readable without JavaScript and has matching SEO and anchors', async () => {
  const page = await openPage({ javaScriptEnabled: false })
  try {
    for (const lang of ['zh', 'en']) {
      const prefix = lang === 'en' ? '/en' : ''
      await page.goto(origin + `${prefix}/blog/`)
      assert.equal(await page.locator('.blog-post-list li').count(), blogPosts.length)
      for (const post of blogPosts) {
        const route = `${prefix}/blog/${post.slug}/`
        const response = await page.goto(origin + route)
        assert.equal(response.status(), 200)
        assert.equal(await page.locator('html').getAttribute('lang'), lang === 'en' ? 'en' : 'zh-CN')
        assert.equal(await page.locator('h1').innerText(), post[lang].title)
        assert.equal(await page.title(), `${post[lang].title} | cjv`)
        assert.equal(await page.locator('meta[name="description"]').getAttribute('content'), post[lang].description)
        assert.equal(await page.locator('link[rel="canonical"]').getAttribute('href'), `https://cjv.zxilly.com${route}`)
        for (const [code, target] of [['zh-CN', ''], ['en', '/en'], ['x-default', '']]) {
          assert.equal(await page.locator(`link[hreflang="${code}"]`).getAttribute('href'), `https://cjv.zxilly.com${target}/blog/${post.slug}/`)
        }
        const schema = JSON.parse(await page.locator('script[type="application/ld+json"]').textContent())
        assert.equal(schema['@type'], 'BlogPosting')
        assert.equal(schema.headline, post[lang].title)
        assert.equal(schema.url, `https://cjv.zxilly.com${route}`)
        assert.ok(await page.locator('.blog-prose pre').count() > 0)
        for (const href of await page.locator('.blog-contents a').evaluateAll(links => links.map(link => link.getAttribute('href')))) {
          assert.equal(await page.locator(href).count(), 1)
        }
        const links = await page.locator('.blog-prose a').evaluateAll(links => links.map(link => link.href))
        assert.ok(links.every(link => !link.includes('cjv.zxilly.dev')))
      }
    }
  } finally { await page.close() }
})

test('Blog navigation loads static article data and preserves the article across languages and history', async () => {
  const page = await openPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
  try {
    await page.goto(origin + '/', { waitUntil: 'networkidle' })
    await page.getByRole('link', { name: '开发实践', exact: true }).click()
    await page.getByRole('link', { name: blogPosts[0].zh.title, exact: true }).click()
    await page.locator('.blog-prose').waitFor()
    await page.getByRole('link', { name: 'English', exact: true }).click()
    await page.getByRole('heading', { level: 1, name: blogPosts[0].en.title, exact: true }).waitFor()
    assert.equal(await page.locator('link[rel="canonical"]').getAttribute('href'), `https://cjv.zxilly.com/en/blog/${blogPosts[0].slug}/`)
    await page.goBack()
    await page.getByRole('heading', { level: 1, name: blogPosts[0].zh.title, exact: true }).waitFor()
    await page.goto(origin + `/en/blog/${blogPosts[0].slug}/index.html`, { waitUntil: 'networkidle' })
    assert.equal(await page.locator('h1').innerText(), blogPosts[0].en.title)
    // The installation homepage needs a fresh edge response for regional defaults.
    const homeRequest = page.waitForRequest(request => request.isNavigationRequest() && request.url() === origin + '/en/')
    await page.getByRole('link', { name: 'Install cjv', exact: true }).click()
    await homeRequest
    await page.getByRole('heading', { level: 1, name: /Cangjie SDK version manager/ }).waitFor()
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

async function openPage(options = {}) {
  const page = await browser.newPage({ locale: 'en-US', reducedMotion: 'reduce', ...options })
  // Keep production analytics out of this local, otherwise network-independent check.
  await page.route('https://trail.learningman.top/**', route => route.fulfill({ status: 200, contentType: 'text/javascript', body: '' }))
  return page
}

test('built homepage detects the real browser and offers its installation path', async t => {
  const page = await openPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
  try {
    await page.goto(origin + '/en/', { waitUntil: 'networkidle' })
    const exposed = await page.evaluate(async () => {
      const info = { userAgent: navigator.userAgent, platform: navigator.platform, hints: null, hintsError: null }
      try {
        info.hints = await navigator.userAgentData?.getHighEntropyValues(['architecture', 'bitness', 'platform']) ?? null
      } catch (error) {
        info.hintsError = String(error)
      }
      return info
    })
    t.diagnostic(JSON.stringify({ host: `${process.platform}/${process.arch}`, ...exposed }))
    // Assert the OS advertised by the real browser. Playwright WebKit on Windows
    // uses a Macintosh UA even when navigator.platform says Win32; its host OS
    // is not a reliable expectation for the page's detected download options.
    const expectedOS = /\bWindows\b/.test(exposed.userAgent) ? 'Windows'
      : /\bMacintosh\b/.test(exposed.userAgent) ? 'macOS'
        : /\bLinux\b/.test(exposed.userAgent) ? 'Linux' : undefined
    assert.ok(expectedOS, `unsupported browser user agent: ${exposed.userAgent}`)
    const hintArch = exposed.hints?.architecture?.toLowerCase()
    let expectedArch = { arm64: 'arm64', aarch64: 'arm64', amd64: 'amd64', x86_64: 'amd64' }[hintArch]
    if (exposed.hints?.bitness === '64') expectedArch ??= { arm: 'arm64', x86: 'amd64' }[hintArch]
    // MacIntel and the Intel Macintosh UA are compatibility values, not proof
    // of an Intel CPU. Other desktop browsers expose explicit 64-bit UA tokens.
    if (!expectedArch && expectedOS !== 'macOS' && exposed.hints?.bitness !== '32') {
      expectedArch = /\b(?:arm64|aarch64)\b/i.test(exposed.userAgent) ? 'arm64'
        : /\b(?:x86_64|amd64|x64|Win64)\b/i.test(exposed.userAgent) ? 'amd64' : undefined
    }
    const windowsARM = expectedOS === 'Windows' && expectedArch === 'arm64'
    const expectedLabel = expectedArch ? `${expectedOS} ${expectedArch === 'arm64' ? 'ARM64' : 'x86_64'}` : 'macOS'
    if (windowsARM) {
      await page.getByText('cjv does not support the Windows arm64 platform yet.', { exact: true }).waitFor()
      assert.equal(await page.getByRole('tablist').count(), 0)
      await page.getByRole('button', { name: 'Show install methods for other platforms', exact: true }).click()
    } else {
      assert.ok(expectedArch || expectedOS === 'macOS', `browser exposes no supported 64-bit architecture: ${JSON.stringify(exposed)}`)
      await page.getByText(`Detected platform: ${expectedLabel}`, { exact: true }).waitFor()
    }
    await page.getByRole('tab', { name: 'Download', exact: true }).click()
    const goos = { Windows: 'windows', macOS: 'darwin', Linux: 'linux' }[expectedOS]
    const downloads = page.getByRole('tabpanel')
    if (expectedOS === 'macOS' && !expectedArch) {
      await downloads.getByText('Choose your Mac chip to download the cjv installer', { exact: true }).waitFor()
      await downloads.locator('a[href="/dl/official/darwin_arm64/cjv-init"]').waitFor()
      await downloads.locator('a[href="/dl/official/darwin_amd64/cjv-init"]').waitFor()
    } else {
      const downloadArch = windowsARM ? 'amd64' : expectedArch
      const binary = goos === 'windows' ? 'cjv-init.exe' : 'cjv-init'
      await downloads.locator(`a[href="/dl/official/${goos}_${downloadArch}/${binary}"]`).waitFor()
      if (windowsARM) {
        await downloads.getByText('Pick the binary for your platform manually:', { exact: true }).waitFor()
        assert.equal(await downloads.locator('a[href^="/dl/official/windows_arm64/"]').count(), 0)
      } else {
        await downloads.getByText(`Detected platform: ${expectedLabel}`, { exact: true }).waitFor()
      }
    }
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('installation content stays visible while font requests are pending', async () => {
  const page = await openPage()
  let releaseFonts
  const pendingFonts = new Promise(resolve => { releaseFonts = resolve })
  let requests = 0
  await page.route(/\.(woff2?|ttf)(\?|$)/, async route => {
    requests++
    await pendingFonts
    await route.abort()
  })
  try {
    await page.goto(origin + '/en/', { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => document.querySelector('#app button') && document.fonts.status === 'loading')
    assert.ok(requests > 0, 'the regression must exercise pending font downloads')
    assert.equal(await page.locator('#app').evaluate(element => getComputedStyle(element).opacity), '1')
    assert.ok(await page.locator('#app a[href*="/book/user-guide/"]').first().isVisible())
  } finally {
    releaseFonts()
    await page.close()
  }
})

test('normal animations settle after tab, mirror, and language changes', async () => {
  const page = await openPage({ reducedMotion: 'no-preference' })
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  try {
    await page.goto(origin + '/en/', { waitUntil: 'networkidle' })
    const otherPlatforms = page.getByRole('button', { name: 'Show install methods for other platforms', exact: true })
    if (await otherPlatforms.count()) await otherPlatforms.click()
    for (const [name, panel] of [['Download', 'download'], ['Source', 'source'], ['Command', 'command']]) {
      await page.getByRole('tab', { name, exact: true }).click()
      await page.waitForFunction(expected => {
        const panels = document.querySelectorAll('[role="tabpanel"]')
        if (panels.length !== 1 || panels[0].id !== `install-panel-${expected}`) return false
        const transform = getComputedStyle(panels[0]).transform
        return transform === 'none' || Math.abs(new DOMMatrixReadOnly(transform).m41) < 0.1
      }, panel)
    }
    await page.getByRole('switch').click()
    await page.getByText('GitCode · mirror', { exact: true }).waitFor()
    for (const [name, lang] of [['中文', 'zh-CN'], ['English', 'en']]) {
      await page.getByRole('link', { name, exact: true }).click()
      await page.waitForFunction(expected => {
        const content = document.querySelector('#app > div')
        return document.documentElement.lang === expected && content && getComputedStyle(content).opacity === '1'
      }, lang)
    }
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('built homepage fits desktop and narrow mobile viewports', async () => {
  const page = await openPage()
  try {
    for (const width of [1280, 360]) {
      await page.setViewportSize({ width, height: 900 })
      await page.goto(origin + '/en/', { waitUntil: 'networkidle' })
      await page.locator('#app button').first().waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), `overflow at ${width}px`)
      if (process.env.CJV_WEB_SCREENSHOTS) {
        await mkdir(process.env.CJV_WEB_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: path.join(process.env.CJV_WEB_SCREENSHOTS, `${browserName}-${width}.png`), fullPage: true })
      }
    }
  } finally {
    await page.close()
  }
})

test('both languages provide visible content, commands and crawlable links without JavaScript', async () => {
  const page = await openPage({ javaScriptEnabled: false })
  try {
    for (const [route, language, heading] of [['/', 'zh-CN', '仓颉 SDK 版本管理器'], ['/en/', 'en', 'Cangjie SDK version manager']]) {
      await page.goto(origin + route)
      assert.equal(await page.locator('html').getAttribute('lang'), language)
      assert.ok((await page.getByRole('heading', { level: 1 }).innerText()).includes(heading))
      assert.ok(await page.getByRole('heading', { level: 2 }).isVisible())
      for (const command of await page.locator('code').all()) assert.ok(await command.isVisible())
      assert.ok((await page.locator('code').allTextContents()).some(value => value.includes('install.ps1')))
      assert.equal(await page.locator('link[rel="canonical"]').getAttribute('href'), 'https://cjv.zxilly.com' + route)
      assert.equal(await page.locator('a[href="/en/"]').count(), 1)
      assert.ok(await page.locator('a[href*="toolchain-file.html"]').isVisible())
    }
  } finally { await page.close() }
})

test('regional .dev HTML hydrates without errors and keeps domain-specific installer commands', async () => {
  const page = await openPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
  await page.route('https://cjv.zxilly.dev/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    const response = await route.fetch({ url: origin + pathname })
    if (pathname === '/' || pathname === '/en/') {
      const injected = homeResponse(await response.text(), 'CN', 'GET', 'cjv.zxilly.dev')
      await route.fulfill({ response, body: await injected.text() })
    } else await route.fulfill({ response })
  })
  try {
    // Saved and browser languages must not override the explicit page URL.
    await page.addInitScript(() => localStorage.setItem('cjv-lang', 'zh'))
    await page.goto('https://cjv.zxilly.dev/en/', { waitUntil: 'networkidle' })
    await page.getByText('GitCode · mirror', { exact: true }).waitFor()
    assert.equal(await page.locator('html').getAttribute('lang'), 'en')
    assert.ok((await page.locator('code').allTextContents()).some(value => value.includes('https://cjv.zxilly.dev/install.')))
    await page.getByRole('link', { name: '中文', exact: true }).click()
    await page.waitForFunction(() => document.documentElement.lang === 'zh-CN')
    assert.equal(new URL(page.url()).pathname, '/')
    await page.goBack()
    await page.waitForFunction(() => document.documentElement.lang === 'en')
    assert.equal(await page.locator('link[rel="canonical"]').getAttribute('href'), 'https://cjv.zxilly.com/en/')
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('explicit index.html URLs hydrate their matching route and retain canonical URLs', async () => {
  const page = await openPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
  try {
    for (const [pathname, language, canonical] of [['/index.html', 'zh-CN', '/'], ['/en/index.html', 'en', '/en/']]) {
      await page.goto(origin + pathname, { waitUntil: 'networkidle' })
      assert.equal(await page.locator('html').getAttribute('lang'), language)
      assert.equal(await page.locator('link[rel="canonical"]').getAttribute('href'), 'https://cjv.zxilly.com' + canonical)
      await page.getByRole('switch').click()
      assert.equal(await page.getByRole('switch').getAttribute('aria-checked'), 'true')
    }
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})
