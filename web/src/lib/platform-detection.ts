import { parseCPU, parseOS } from 'ua-parser-modern'
import {
  findInstallation, MAC_COMMAND, HARMONY_COMMAND, METHODS, SOURCE_METHOD, otherMethodsFor,
  MANUAL_DOWNLOAD, MAC_DOWNLOAD, HARMONY_DOWNLOAD,
  type BinaryInfo, type CommandInfo, type InstallMethod,
  type DetectedDownload, type ArchitectureDownload, type ManualDownload,
} from './platform-catalog'

const UNSUPPORTED = new Set(['iOS', 'Android'])

export interface ReadyInfo extends CommandInfo {
  harmony?: HarmonyInfo
  // A ready result is by definition supported, so it never carries an unsupported
  // reason. Declaring the field as `never` keeps `info.reason` accessible across the
  // ReadyInfo | BasicInfo union without widening what a ready result may hold.
  reason?: never
}

// Why a visitor is unsupported, so the UI can tailor its advice:
//   'mobile' — a phone/tablet OS (iOS/Android); ask them to use a desktop.
//   'harmony' — the browser cannot establish native terminal/architecture support.
//   'arch'   — a known desktop OS whose CPU architecture has no prebuilt binary
//              (e.g. Windows arm64); suggest the amd64 build or a manual download.
export type UnsupportedReason = 'mobile' | 'arch' | 'harmony'

export interface HarmonyInfo {
  // Keep the UA's version namespace; OpenHarmony is not a marketing version.
  family: 'OpenHarmony' | 'HarmonyOS'
  version?: string
  device: 'phone' | 'tablet' | 'convertible-tablet' | 'pc' | 'unknown'
}

export interface BasicInfo {
  label: string
  reason?: UnsupportedReason
  harmony?: HarmonyInfo
}

export type PlatformState = 'ready' | 'unsupported' | 'unknown'

interface CommonResult {
  methods: InstallMethod[]
  otherMethods: InstallMethod[]
  sourceMethod: InstallMethod
}

export type PlatformResult = CommonResult & (
  // When the OS is supported but its native CPU is unknown, the installer resolves
  // the architecture and manual downloads offer a choice for that OS.
  | { state: 'ready'; info: ReadyInfo; download: DetectedDownload | ArchitectureDownload }
  | { state: 'unsupported' | 'unknown'; info: BasicInfo; download: ManualDownload }
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

export function computePlatformResult(os: string, arch: string): PlatformResult {
  const goos = normalizeOS(os)
  const goarch = normalizeArch(arch)
  const entry = goos && goarch ? findInstallation(goos, goarch) : undefined
  const knownDesktopOS = goos !== null
  const hasArch = arch.trim() !== ''
  const common: CommonResult = {
    methods: METHODS,
    otherMethods: METHODS,
    sourceMethod: SOURCE_METHOD,
  }
  if (entry) {
    const { label, hint, command, mirrorCommand, warning } = entry
    const info: ReadyInfo = { label, hint, command, mirrorCommand, warning }
    return {
      ...common,
      otherMethods: otherMethodsFor(entry.goos),
      state: 'ready',
      info,
      download: entry.download,
    }
  }
  // macOS without a detectable arch (Safari/Firefox hide it): install.sh resolves the
  // arch itself, so we stay 'ready' with no specific binary rather than giving up.
  if (goos === 'darwin' && !hasArch) {
    const info = MAC_COMMAND
    return {
      ...common,
      otherMethods: otherMethodsFor('darwin'),
      state: 'ready',
      info,
      download: MAC_DOWNLOAD,
    }
  }
  const isMobile = UNSUPPORTED.has(os)
  const isHarmony = os === 'HarmonyOS'
  const isUnsupportedArch = knownDesktopOS && hasArch
  const state: 'unsupported' | 'unknown' =
    isMobile || isHarmony || isUnsupportedArch ? 'unsupported' : 'unknown'
  return {
    ...common,
    state,
    info: {
      label:
        isMobile || isHarmony ? os
        : isUnsupportedArch ? `${displayOS(os)} ${arch}`
        : knownDesktopOS ? `${displayOS(os)} 未知架构`
        : '未知平台',
      // Only set on the 'unsupported' branch; 'unknown' carries no reason.
      reason: isHarmony ? 'harmony' : isMobile ? 'mobile' : isUnsupportedArch ? 'arch' : undefined,
    },
    download: MANUAL_DOWNLOAD,
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

// Harmony UAs can include Android or Windows compatibility tokens. Explicit OS
// tokens take precedence over those tokens, Client Hints, and iPad heuristics.
function parseHarmony(ua: string = ''): HarmonyInfo | undefined {
  const match = /\b(OpenHarmony|HarmonyOS)\b(?:[ /](\d+(?:\.\d+)*))?/i.exec(ua)
  if (!match) return undefined
  const tablet = /\bTablet\b/i.test(ua)
  return {
    family: match[1].toLowerCase() === 'openharmony' ? 'OpenHarmony' : 'HarmonyOS',
    version: match[2],
    // The supplied convertible sample uses Tablet together with Windows NT.
    // Its Win64/x64 tokens do not establish the device's native CPU architecture.
    device: tablet ? (/\bWindows NT\b/i.test(ua) ? 'convertible-tablet' : 'tablet')
      : /\bPhone\b|\bMobile\b/i.test(ua) ? 'phone'
      : /\bPC\b|\bWindows NT\b/i.test(ua) ? 'pc' : 'unknown',
  }
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
  const harmony = parseHarmony(input.userAgent)
  if (harmony) {
    const label = !harmony.version ? 'HarmonyOS'
      : harmony.family === 'OpenHarmony' ? `HarmonyOS (OpenHarmony ${harmony.version})`
      : `HarmonyOS ${harmony.version}`
    // Compatibility UA and Client Hints cannot establish the native CPU.
    if (harmony.device === 'pc' || harmony.device === 'convertible-tablet') {
      return {
        ...computePlatformResult('HarmonyOS', ''),
        otherMethods: otherMethodsFor('openharmony'),
        state: 'ready',
        info: {
          ...HARMONY_COMMAND, label, harmony,
        },
        download: HARMONY_DOWNLOAD,
      }
    }
    return {
      ...computePlatformResult('HarmonyOS', ''),
      state: 'unsupported',
      info: {
        label,
        reason: 'harmony',
        harmony,
      },
      download: MANUAL_DOWNLOAD,
    }
  }
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
