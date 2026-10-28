import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { chromium, firefox, webkit } from 'playwright'
import { preview } from 'vite'

let server
let browser
let origin
const browserName = process.env.PLAYWRIGHT_BROWSER || 'chromium'

before(async () => {
  server = await preview({
    root: fileURLToPath(new URL('..', import.meta.url)),
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
    await page.goto(origin, { waitUntil: 'networkidle' })
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
    await page.goto(origin, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => document.querySelector('#app button') && document.fonts.status === 'loading')
    assert.ok(requests > 0, 'the regression must exercise pending font downloads')
    assert.equal(await page.locator('#app').evaluate(element => getComputedStyle(element).opacity), '1')
    assert.ok(await page.locator('#app a[href*="/book/user-guide/"]').isVisible())
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
    await page.goto(origin, { waitUntil: 'networkidle' })
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
      await page.getByRole('button', { name, exact: true }).click()
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
      await page.goto(origin, { waitUntil: 'networkidle' })
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
