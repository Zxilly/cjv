import { parseCPU, parseOS } from 'ua-parser-modern'

const BASE = 'https://cjv.zxilly.dev'

const UNSUPPORTED = new Set(['iOS', 'Android', 'HarmonyOS'])

interface InstallInfo {
  label: string
  hint?: string
  command?: string
  warning?: string
}

interface InstallMethod {
  label: string
  command: string
}

type PlatformState = 'ready' | 'unsupported' | 'unknown'

function detectPlatform() {
  const os = parseOS().name || ''
  const arch = parseCPU().architecture || 'amd64'
  return { os, arch }
}

function getInstallInfo(os: string, arch: string): InstallInfo {
  if (UNSUPPORTED.has(os)) return { label: os }
  if (os === 'Mac OS' || os === 'macOS') {
    return arch === 'arm64'
      ? { label: 'macOS ARM64', hint: '在终端中运行：', command: `curl -sSf ${BASE}/install.sh | sh` }
      : { label: 'macOS x86_64', hint: '在终端中运行：', command: `curl -sSf ${BASE}/install.sh | sh`, warning: '部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。' }
  }
  if (os === 'Windows') return { label: 'Windows x86_64', hint: '在 PowerShell 中运行：', command: `irm ${BASE}/install.ps1 | iex` }
  if (os === 'Linux') return { label: `Linux ${arch === 'arm64' ? 'ARM64' : 'x86_64'}`, hint: '在终端中运行：', command: `curl -sSf ${BASE}/install.sh | sh` }
  return { label: '未知平台' }
}

export function usePlatform() {
  const { os, arch } = detectPlatform()
  const info = getInstallInfo(os, arch)
  const state: PlatformState = !info.command
    ? (UNSUPPORTED.has(os) ? 'unsupported' : 'unknown')
    : 'ready'

  const methods: InstallMethod[] = [
    { label: 'Linux / macOS', command: `curl -sSf ${BASE}/install.sh | sh` },
    { label: 'Windows (PowerShell)', command: `irm ${BASE}/install.ps1 | iex` },
    { label: '从源码编译', command: 'go install github.com/Zxilly/cjv/cmd/cjv@latest' },
  ]

  const mirrorMethods: InstallMethod[] = [
    { label: 'Linux / macOS', command: `curl -sSf ${BASE}/install.sh | sh -s -- --mirror` },
    { label: 'Windows (PowerShell)', command: `$env:CJV_MIRROR = "1"; irm ${BASE}/install.ps1 | iex` },
  ]

  return { state, info, methods, mirrorMethods }
}
