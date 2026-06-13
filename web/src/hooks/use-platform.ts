import { useEffect, useState } from 'react'
import { msg } from '@lingui/core/macro'
import type { MessageDescriptor } from '@lingui/core'
import { parseCPU, parseOS } from 'ua-parser-modern'
import { SUPPORTED_PLATFORMS, type SupportedPlatform } from '../generated/platforms'

const BASE = 'https://cjv.zxilly.dev'
const REPO = 'https://github.com/Zxilly/cjv'
const GITCODE = 'https://gitcode.com/Zxilly/cjv'
const DL_BASE = '/dl'

const UNSUPPORTED = new Set(['iOS', 'Android', 'HarmonyOS'])
const MAC_X86_WARNING = msg`部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。`

export interface ReadyInfo {
  label: string
  hint: MessageDescriptor
  command: string
  mirrorCommand: string
  warning?: MessageDescriptor
  // A ready result is by definition supported, so it never carries an unsupported
  // reason. Declaring the field as `never` keeps `info.reason` accessible across the
  // ReadyInfo | BasicInfo union without widening what a ready result may hold.
  reason?: never
}

// Why a visitor is unsupported, so the UI can tailor its advice:
//   'mobile' — a phone/tablet OS (iOS/Android/HarmonyOS); ask them to use a desktop.
//   'arch'   — a known desktop OS whose CPU architecture has no prebuilt binary
//              (e.g. Windows arm64); suggest the amd64 build or a manual download.
export type UnsupportedReason = 'mobile' | 'arch'

export interface BasicInfo {
  label: string
  reason?: UnsupportedReason
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
  warning?: MessageDescriptor
}

export type PlatformState = 'ready' | 'unsupported' | 'unknown'

interface CommonResult {
  methods: InstallMethod[]
  otherMethods: InstallMethod[]
  sourceMethod: InstallMethod
  allBinaries: BinaryInfo[]
}

export type PlatformResult = CommonResult & (
  // In the 'ready' state `binary` is null only when the OS is known but the CPU
  // architecture is hidden (macOS on Safari/Firefox): install.sh still installs the
  // right build, but there is no single binary to offer — the UI shows an arch choice.
  | { state: 'ready'; info: ReadyInfo; binary: BinaryInfo | null }
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

const SH_HINT = msg`在终端中运行：`
const PS_HINT = msg`在 PowerShell 中运行：`

type PlatformEntry = SupportedPlatform & {
  label: string
  hint: MessageDescriptor
  command: string
  mirrorCommand: string
  warning?: MessageDescriptor
}

type SupportedPlatformKey<T extends SupportedPlatform = SupportedPlatform> =
  T extends { goos: infer GOOS extends string; goarch: infer GOARCH extends string }
    ? `${GOOS}_${GOARCH}`
    : never

const PLATFORM_PRESENTATION: Record<SupportedPlatformKey, Omit<PlatformEntry, 'goos' | 'goarch'>> = {
  windows_amd64: { label: 'Windows x86_64', hint: PS_HINT, command: PS_CMD, mirrorCommand: PS_MIRROR_CMD },
  darwin_arm64: { label: 'macOS ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  darwin_amd64: { label: 'macOS x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD, warning: MAC_X86_WARNING },
  linux_amd64: { label: 'Linux x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  linux_arm64: { label: 'Linux ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
}

const PLATFORM_ORDER: SupportedPlatformKey[] = [
  'windows_amd64',
  'darwin_arm64',
  'darwin_amd64',
  'linux_amd64',
  'linux_arm64',
]

const GENERATED_PLATFORM_BY_KEY = new Map(SUPPORTED_PLATFORMS.map(p => [`${p.goos}_${p.goarch}`, p] as const))

const PLATFORMS: PlatformEntry[] = PLATFORM_ORDER.map(key => {
  const platform = GENERATED_PLATFORM_BY_KEY.get(key)
  if (!platform) throw new Error(`missing generated platform ${key}`)
  return { ...platform, ...PLATFORM_PRESENTATION[key] }
})

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

const UNIX_METHOD: InstallMethod = { label: 'Linux / macOS', command: SH_CMD, mirrorCommand: SH_MIRROR_CMD }
const WINDOWS_METHOD: InstallMethod = { label: 'Windows (PowerShell)', command: PS_CMD, mirrorCommand: PS_MIRROR_CMD }

const METHODS: InstallMethod[] = [UNIX_METHOD, WINDOWS_METHOD]

// install.sh installs on both Linux and macOS, so for a non-Unix visitor we keep the
// single combined "Linux / macOS" row. But when the visitor IS on Linux or macOS, that
// combined row is their own platform — filtering it out would also drop the sibling Unix
// OS from 其他平台. So we name the remaining sibling explicitly instead.
function otherMethodsFor(detected: BinaryInfo['goos']): InstallMethod[] {
  if (detected === 'windows') return [UNIX_METHOD]
  const sibling: InstallMethod = {
    label: detected === 'linux' ? 'macOS' : 'Linux',
    command: SH_CMD,
    mirrorCommand: SH_MIRROR_CMD,
  }
  return [sibling, WINDOWS_METHOD]
}

const SOURCE_METHOD: InstallMethod = {
  label: '从源码编译',
  command: 'go install github.com/Zxilly/cjv/cmd/cjv@latest',
  mirrorCommand: 'GOPROXY=https://goproxy.cn,direct go install github.com/Zxilly/cjv/cmd/cjv@latest',
}

export function computePlatformResult(os: string, arch: string): PlatformResult {
  const entry = findEntry(os, arch)
  const goos = normalizeOS(os)
  const knownDesktopOS = goos !== null
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
      otherMethods: otherMethodsFor(entry.goos),
      state: 'ready',
      info,
      binary: binaryForEntry(entry),
    }
  }
  // macOS without a detectable arch (Safari/Firefox hide it): install.sh resolves the
  // arch itself, so we stay 'ready' with no specific binary rather than giving up.
  if (goos === 'darwin' && !hasArch) {
    const info: ReadyInfo = { label: 'macOS', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD }
    return {
      ...common,
      otherMethods: otherMethodsFor('darwin'),
      state: 'ready',
      info,
      binary: null,
    }
  }
  const isMobile = UNSUPPORTED.has(os)
  const isUnsupportedArch = knownDesktopOS && hasArch
  const state: 'unsupported' | 'unknown' =
    isMobile || isUnsupportedArch ? 'unsupported' : 'unknown'
  return {
    ...common,
    state,
    info: {
      label:
        isMobile ? os
        : isUnsupportedArch ? `${displayOS(os)} ${arch}`
        : knownDesktopOS ? `${displayOS(os)} 未知架构`
        : '未知平台',
      // Only set on the 'unsupported' branch; 'unknown' carries no reason.
      reason: isMobile ? 'mobile' : isUnsupportedArch ? 'arch' : undefined,
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

// ua-parser-modern only emits OS name 'HarmonyOS' when the UA carries both 'android' and
// 'harmonyos'. HarmonyOS NEXT drops the Android token, so it parses as undefined or
// 'Linux' and would otherwise land in 'unknown' (or worse, 'Linux'), showing a Harmony
// phone user a wall of desktop commands. The 'harmonyos'/'openharmony' tokens only appear
// in genuine Harmony UAs, so matching them in the raw UA is a reliable override.
function parseBrowserOS(input: BrowserPlatformInput): string {
  if (/harmonyos|openharmony/i.test(input.userAgent ?? '')) return 'HarmonyOS'
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
