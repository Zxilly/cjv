import { createHash } from 'node:crypto'
import { readFile, readdir } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export async function deploymentMessage(outputDir) {
  const hash = createHash('sha256')
  async function visit(relativeDir) {
    const entries = await readdir(path.join(outputDir, relativeDir), { withFileTypes: true })
    entries.sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    for (const entry of entries) {
      const relativePath = relativeDir ? `${relativeDir}/${entry.name}` : entry.name
      if (entry.isDirectory()) {
        await visit(relativePath)
      } else if (entry.isFile()) {
        const content = await readFile(path.join(outputDir, relativePath))
        hash.update(`${relativePath}\0${content.length}\0`)
        hash.update(content)
      } else {
        throw new Error(`Unsupported build output entry: ${relativePath}`)
      }
    }
  }
  await visit('')
  return `cjv-edge-sha256:${hash.digest('hex')}`
}

export function matchesActiveDeployment(deployments, message) {
  if (!Array.isArray(deployments)) throw new Error('Cloudflare response is missing deployments')
  const current = [...deployments].sort((a, b) => Date.parse(b.created_on) - Date.parse(a.created_on))[0]
  return current?.annotations?.['workers/message'] === message
    && current.versions?.length === 1
    && current.versions[0].percentage === 100
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [outputDir, deploymentsFile] = process.argv.slice(2)
  if (!outputDir || !deploymentsFile) throw new Error('Usage: node deployment-state.mjs <build output> <deployments JSON>')
  const message = await deploymentMessage(outputDir)
  const { deployments } = JSON.parse(await readFile(deploymentsFile, 'utf8'))
  console.log(`message=${message}`)
  console.log(`changed=${!matchesActiveDeployment(deployments, message)}`)
}
