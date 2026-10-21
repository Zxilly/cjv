import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { homeResponse } from './home-response.mjs'

export async function buildEdge(provider, siteDir, outputDir) {
  if (!['edgeone', 'cloudflare'].includes(provider)) throw new Error(`Unknown provider: ${provider}`)
  // The CLI recognizes prebuilt functions only when the upload root is .edgeone.
  if (provider === 'edgeone' && path.basename(outputDir) !== '.edgeone') {
    throw new Error('EdgeOne prebuilt output directory must be named .edgeone')
  }
  await mkdir(outputDir, { recursive: true })

  if (provider === 'edgeone') {
    const html = await readFile(path.join(siteDir, 'index.html'), 'utf8')
    if (!html.includes('<html ')) throw new Error('Missing HTML element in the built homepage')
    const source = `import 'hooks';\nconst html = ${JSON.stringify(html)};\n${homeResponse.toString()}\n`
    await cp(siteDir, path.join(outputDir, 'assets'), { recursive: true })
    await mkdir(path.join(outputDir, 'edge-functions'), { recursive: true })
    await writeFile(path.join(outputDir, 'edge-functions/index.js'), source + `
addEventListener('fetch', (event, hookCtx) => {
  const pathname = new URL(event.request.url).pathname;
  if (pathname !== '/' && pathname !== '/index.html') {
    const forward = hookCtx?.fetch || globalThis.fetch;
    event.respondWith(forward(event.request));
    return;
  }
  event.respondWith(homeResponse(html, event.request.eo?.geo?.countryCodeAlpha2, event.request.method));
});
`)
    await writeFile(path.join(outputDir, 'edge-functions/config.json'), JSON.stringify({
      routes: [{ src: '^/(index\\.html)?$' }],
    }, null, 2) + '\n')
    await cp(new URL('./edgeone/routes.json', import.meta.url), path.join(outputDir, 'routes.json'))
  } else {
    const workerDir = path.join(outputDir, 'workers/default')
    await mkdir(path.join(workerDir, 'bundle'), { recursive: true })
    const worker = await readFile(new URL('./cloudflare/worker.mjs', import.meta.url), 'utf8')
    await writeFile(path.join(workerDir, 'bundle/worker.mjs'), worker.replace(
      "import { homeResponse } from '../home-response.mjs'", homeResponse.toString(),
    ))
    await cp(new URL('./cloudflare/worker.config.json', import.meta.url), path.join(workerDir, 'worker.config.json'))
    await writeFile(path.join(outputDir, 'config.json'), JSON.stringify({ buildContext: { isPreview: false } }) + '\n')
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [provider, siteDir, outputDir] = process.argv.slice(2)
  if (!provider || !siteDir || !outputDir) throw new Error('Usage: node deployment/build-edge.mjs <edgeone|cloudflare> <site> <output>')
  await buildEdge(provider, siteDir, outputDir)
}
