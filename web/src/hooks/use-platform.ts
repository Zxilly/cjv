import { useEffect, useState } from 'react'
import { parseCPU, parseOS } from 'ua-parser-modern'

const BASE = 'https://cjv.zxilly.dev'
const REPO = 'https://github.com/Zxilly/cjv'
const GITCODE = 'https://gitcode.com/Zxilly/cjv'
const DL_BASE = '/dl'

const UNSUPPORTED = new Set(['iOS', 'Android', 'HarmonyOS'])
const MAC_X86_WARNING = '部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。'

export interface ReadyInfo {
  label: string
  hint: string
  command: string
  mirrorCommand: string
  warning?: string
}

export interface BasicInfo {
  label: string
}

export interface InstallMethod {
  label: string
  command: string
  mirrorCommand?: string
}

export interface BinaryInfo {
  label: string
  goos: 'linux' | 'darwin' | 'windows'
  goarch: 'amd64' | 'arm64'
  binaryName: string
  officialUrl: string
  mirrorUrl: string
  officialReleaseUrl: string
  mirrorReleaseUrl: string
  warning?: string
}

export type PlatformState = 'ready' | 'unsupported' | 'unknown'

interface CommonResult {
  methods: InstallMethod[]
  otherMethods: InstallMethod[]
  sourceMethod: InstallMethod
  allBinaries: BinaryInfo[]
}

export type PlatformResult = CommonResult & (
  | { state: 'ready'; info: ReadyInfo; binary: BinaryInfo }
  | { state: 'unsupported' | 'unknown'; info: BasicInfo; binary: null }
)

interface UserAgentDataLike {
  architecture?: string
  bitness?: string
  mobile?: boolean
  platform?: string
  getHighEntropyValues?: (hints: string[]) => Promise<UserAgentDataLike>
}

export interface BrowserPlatformInput {
  maxTouchPoints?: number
  platform?: string
  userAgent?: string
  userAgentData?: UserAgentDataLike | null
}

const SH_CMD = `curl -sSf ${BASE}/install.sh | sh`
const SH_MIRROR_CMD = `curl -sSf ${BASE}/install.sh | sh -s -- --mirror`
const PS_CMD = `irm ${BASE}/install.ps1 | iex`
const PS_MIRROR_CMD = `$env:CJV_MIRROR = "1"; irm ${BASE}/install.ps1 | iex`

const SH_HINT = '在终端中运行：'
const PS_HINT = '在 PowerShell 中运行：'

interface PlatformEntry {
  goos: BinaryInfo['goos']
  goarch: BinaryInfo['goarch']
  label: string
  hint: string
  command: string
  mirrorCommand: string
  warning?: string
}

const PLATFORMS: PlatformEntry[] = [
  { goos: 'windows', goarch: 'amd64', label: 'Windows x86_64', hint: PS_HINT, command: PS_CMD, mirrorCommand: PS_MIRROR_CMD },
  { goos: 'darwin', goarch: 'arm64', label: 'macOS ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  { goos: 'darwin', goarch: 'amd64', label: 'macOS x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD, warning: MAC_X86_WARNING },
  { goos: 'linux', goarch: 'amd64', label: 'Linux x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  { goos: 'linux', goarch: 'arm64', label: 'Linux ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
]

function toBinaryInfo(p: PlatformEntry): BinaryInfo {
  const isWin = p.goos === 'windows'
  const ext = isWin ? '.zip' : '.tar.gz'
  const binaryName = isWin ? 'cjv-init.exe' : 'cjv-init'
  const platform = `${p.goos}_${p.goarch}`
  return {
    label: p.label,
    goos: p.goos,
    goarch: p.goarch,
    binaryName,
    officialUrl: `${DL_BASE}/official/${platform}/${binaryName}`,
    mirrorUrl: `${DL_BASE}/mirror/${platform}/${binaryName}`,
    officialReleaseUrl: `${REPO}/releases/latest/download/cjv_${platform}${ext}`,
    mirrorReleaseUrl: `${GITCODE}/releases/latest/download/cjv-mirror_${platform}${ext}`,
    warning: p.warning,
  }
}

function toReadyInfo(p: PlatformEntry): ReadyInfo {
  return { label: p.label, hint: p.hint, command: p.command, mirrorCommand: p.mirrorCommand, warning: p.warning }
}

function normalizeOS(os: string): BinaryInfo['goos'] | null {
  return (
    os === 'Windows' ? 'windows'
    : os === 'Mac OS' || os === 'macOS' ? 'darwin'
    : os === 'Linux' ? 'linux'
    : null
  )
}

function normalizeArch(arch: string): BinaryInfo['goarch'] | null {
  if (arch === 'amd64' || arch === 'x86_64') return 'amd64'
  if (arch === 'arm64' || arch === 'aarch64') return 'arm64'
  return null
}

function normalizeClientHintOS(platform: string | undefined): string {
  if (!platform || platform === 'Unknown') return ''
  if (/^macos$/i.test(platform)) return 'Mac OS'
  if (/^chrome os$/i.test(platform)) return 'Chromium OS'
  return platform
}

function normalizeClientHintArch(architecture: string | undefined, bitness: string | undefined): string {
  const arch = architecture?.toLowerCase()
  if (!arch) return ''
  if (arch === 'x86' && bitness === '64') return 'amd64'
  if (arch === 'x86' && bitness === '32') return 'ia32'
  if (arch === 'arm' && bitness === '64') return 'arm64'
  if (arch === 'arm' && !bitness) return ''
  return architecture || ''
}

function detectArchFromUserAgent(ua: string | undefined): string {
  if (!ua) return ''
  if (/\b(?:arm64|aarch64)\b/i.test(ua)) return 'arm64'
  return ''
}

function displayOS(os: string): string {
  const goos = normalizeOS(os)
  if (goos === 'darwin') return 'macOS'
  return os
}

function findEntry(os: string, arch: string): PlatformEntry | undefined {
  const goos = normalizeOS(os)
  const goarch = normalizeArch(arch)
  if (!goos || !goarch) return undefined
  return PLATFORMS.find(p => p.goos === goos && p.goarch === goarch)
}

const ALL_BINARIES: BinaryInfo[] = PLATFORMS.map(toBinaryInfo)

function binaryForEntry(entry: PlatformEntry): BinaryInfo {
  const binary = ALL_BINARIES.find(b => b.goos === entry.goos && b.goarch === entry.goarch)
  if (!binary) throw new Error(`missing binary for ${entry.goos}_${entry.goarch}`)
  return binary
}

const METHODS: InstallMethod[] = [
  { label: 'Linux / macOS', command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  { label: 'Windows (PowerShell)', command: PS_CMD, mirrorCommand: PS_MIRROR_CMD },
]

const SOURCE_METHOD: InstallMethod = {
  label: '从源码编译',
  command: 'go install github.com/Zxilly/cjv/cmd/cjv@latest',
}

export function computePlatformResult(os: string, arch: string): PlatformResult {
  const entry = findEntry(os, arch)
  const knownDesktopOS = normalizeOS(os) !== null
  const hasArch = arch.trim() !== ''
  const common: CommonResult = {
    methods: METHODS,
    otherMethods: METHODS,
    sourceMethod: SOURCE_METHOD,
    allBinaries: ALL_BINARIES,
  }
  if (entry) {
    const info = toReadyInfo(entry)
    return {
      ...common,
      otherMethods: METHODS.filter(m => m.command !== info.command),
      state: 'ready',
      info,
      binary: binaryForEntry(entry),
    }
  }
  const state: 'unsupported' | 'unknown' =
    UNSUPPORTED.has(os) || (knownDesktopOS && hasArch) ? 'unsupported' : 'unknown'
  return {
    ...common,
    state,
    info: {
      label:
        UNSUPPORTED.has(os) ? os
        : knownDesktopOS && hasArch ? `${displayOS(os)} ${arch}`
        : knownDesktopOS ? `${displayOS(os)} 未知架构`
        : '未知平台',
    },
    binary: null,
  }
}

function readBrowserPlatformInput(): BrowserPlatformInput {
  if (typeof window === 'undefined') return {}
  const nav = window.navigator as Navigator & { userAgentData?: UserAgentDataLike }
  return {
    maxTouchPoints: nav.maxTouchPoints,
    platform: nav.platform,
    userAgent: nav.userAgent,
    userAgentData: nav.userAgentData,
  }
}

function isIPadOSDesktopMode(input: BrowserPlatformInput): boolean {
  return input.platform === 'MacIntel' && (input.maxTouchPoints || 0) > 1
}

function parseBrowserOS(input: BrowserPlatformInput): string {
  return parseOS(input.userAgent).name || normalizeClientHintOS(input.userAgentData?.platform)
}

function parseBrowserArch(input: BrowserPlatformInput): string {
  return normalizeClientHintArch(input.userAgentData?.architecture, input.userAgentData?.bitness)
    || detectArchFromUserAgent(input.userAgent)
    || parseCPU(input.userAgent).architecture
    || ''
}

export function computeBrowserPlatformResult(input: BrowserPlatformInput = readBrowserPlatformInput()): PlatformResult {
  if (isIPadOSDesktopMode(input)) return computePlatformResult('iOS', 'arm64')
  return computePlatformResult(parseBrowserOS(input), parseBrowserArch(input))
}

export async function detectBrowserPlatformResult(
  input: BrowserPlatformInput = readBrowserPlatformInput(),
): Promise<PlatformResult> {
  const uaData = input.userAgentData
  if (!uaData?.getHighEntropyValues) return computeBrowserPlatformResult(input)

  try {
    const highEntropy = await uaData.getHighEntropyValues(['architecture', 'bitness', 'platform'])
    return computeBrowserPlatformResult({
      ...input,
      userAgentData: { ...uaData, ...highEntropy },
    })
  } catch {
    return computeBrowserPlatformResult(input)
  }
}

// Two results are equivalent for rendering purposes when their user-visible
// fields match. Comparing these lets usePlatform skip a redundant re-render
// when the async detection produces a structurally identical (but freshly
// allocated) result — e.g. on browsers that expose no UA Client Hints and so
// cannot refine the initial guess.
function samePlatformResult(a: PlatformResult, b: PlatformResult): boolean {
  return (
    a.state === b.state
    && a.info.label === b.info.label
    && a.binary?.goos === b.binary?.goos
    && a.binary?.goarch === b.binary?.goarch
  )
}

export function usePlatform(input?: BrowserPlatformInput) {
  const [platform, setPlatform] = useState(() => computeBrowserPlatformResult(input))

  // Depend on the primitive fields rather than the input object's identity, so
  // a caller passing a fresh object literal each render does not re-trigger the
  // async detection on every render.
  const { maxTouchPoints, platform: navPlatform, userAgent, userAgentData } = input ?? {}

  useEffect(() => {
    let active = true
    detectBrowserPlatformResult(input).then(next => {
      if (!active) return
      setPlatform(prev => (samePlatformResult(prev, next) ? prev : next))
    })
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- input is consumed via its primitive fields above
  }, [maxTouchPoints, navPlatform, userAgent, userAgentData])

  return platform
}
