const BASE = 'https://cjv.zxilly.dev'

const UNSUPPORTED: Record<string, string> = {
  ios: 'iOS',
  android: 'Android',
  harmonyos: 'HarmonyOS',
}

interface InstallInfo {
  label: string
  hint?: string
  command?: string
}

interface InstallMethod {
  label: string
  command: string
}

type PlatformState = 'ready' | 'unsupported' | 'unknown'

function detectPlatform() {
  const ua = navigator.userAgent.toLowerCase()
  const platform = (navigator.platform || '').toLowerCase()

  if (/iphone|ipad|ipod/.test(ua)) return { os: 'ios', arch: 'arm64' }
  if (/android/.test(ua)) return { os: 'android', arch: 'arm64' }
  if (/harmonyos|hmos/.test(ua)) return { os: 'harmonyos', arch: 'arm64' }

  let os = 'unknown'
  if (ua.includes('win') || platform.includes('win')) os = 'windows'
  else if (ua.includes('mac') || platform.includes('mac')) os = 'darwin'
  else if (ua.includes('linux') || platform.includes('linux')) os = 'linux'

  if (os === 'unknown') {
    const av = (navigator.appVersion || '').toLowerCase()
    const oc = ((navigator as Record<string, unknown>).oscpu as string || '').toLowerCase()
    if (av.includes('win') || oc.includes('win')) os = 'windows'
    else if (av.includes('mac') || oc.includes('mac')) os = 'darwin'
    else if (av.includes('linux') || oc.includes('linux')) os = 'linux'
  }

  let arch = 'amd64'
  if (navigator.userAgentData?.architecture?.toLowerCase() === 'arm') arch = 'arm64'
  else if (/arm64|aarch64/.test(ua) || platform.includes('arm')) arch = 'arm64'

  return { os, arch }
}

function getInstallInfo(os: string, arch: string): InstallInfo {
  if (UNSUPPORTED[os]) return { label: UNSUPPORTED[os] }
  if (os === 'windows') return { label: 'Windows x86_64', hint: '在 PowerShell 中运行：', command: `irm ${BASE}/install.ps1 | iex` }
  if (os === 'darwin' || os === 'linux') {
    const ol = os === 'darwin' ? 'macOS' : 'Linux'
    return { label: `${ol} ${arch === 'arm64' ? 'ARM64' : 'x86_64'}`, hint: '在终端中运行：', command: `curl -sSf ${BASE}/install.sh | sh` }
  }
  return { label: '未知平台' }
}

export function usePlatform() {
  const { os, arch } = detectPlatform()
  const info = getInstallInfo(os, arch)
  const state: PlatformState = UNSUPPORTED[os]
    ? 'unsupported'
    : os === 'unknown'
      ? 'unknown'
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
