import { msg } from '@lingui/core/macro'
import type { MessageDescriptor } from '@lingui/core'
import { SUPPORTED_PLATFORMS, type SupportedPlatform } from '../generated/platforms'

const BASE = typeof window !== 'undefined' && window.location.hostname === 'cjv.zxilly.dev'
  ? 'https://cjv.zxilly.dev'
  : 'https://cjv.zxilly.com'
const REPO = 'https://github.com/Zxilly/cjv'
const GITCODE = 'https://gitcode.com/Zxilly/cjv'
const DL_BASE = '/dl'

const MAC_X86_WARNING = msg`部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。`

export interface InstallMethod {
  label: string
  command: string
  mirrorCommand?: string
}

export interface BinaryInfo {
  label: string
  goos: SupportedPlatform['goos']
  goarch: 'amd64' | 'arm64'
  binaryName: string
  officialUrl: string
  mirrorUrl: string
  officialReleaseUrl: string
  mirrorReleaseUrl: string
  warning?: MessageDescriptor
}

export interface CommandInfo {
  label: string
  hint: MessageDescriptor
  command: string
  mirrorCommand: string
  warning?: MessageDescriptor
}

const SH_CMD = `curl -sSf ${BASE}/install.sh | sh`
const SH_MIRROR_CMD = `curl -sSf ${BASE}/install.sh | sh -s -- --mirror`
const PS_CMD = `irm ${BASE}/install.ps1 | iex`
const PS_MIRROR_CMD = `$env:CJV_MIRROR = "1"; irm ${BASE}/install.ps1 | iex`

const SH_HINT = msg`在终端中运行：`
const PS_HINT = msg`在 PowerShell 中运行：`

type PlatformEntry = SupportedPlatform & CommandInfo

type SupportedPlatformKey<T extends SupportedPlatform = SupportedPlatform> =
  T extends { goos: infer GOOS extends string; goarch: infer GOARCH extends string }
    ? `${GOOS}_${GOARCH}`
    : never

// Property order is the display order for every download list and chooser.
const PLATFORM_PRESENTATION: Record<SupportedPlatformKey, Omit<PlatformEntry, 'goos' | 'goarch'>> = {
  windows_amd64: { label: 'Windows x86_64', hint: PS_HINT, command: PS_CMD, mirrorCommand: PS_MIRROR_CMD },
  darwin_arm64: { label: 'macOS ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  darwin_amd64: { label: 'macOS x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD, warning: MAC_X86_WARNING },
  linux_amd64: { label: 'Linux x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  linux_arm64: { label: 'Linux ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  openharmony_amd64: { label: 'HarmonyOS x86_64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
  openharmony_arm64: { label: 'HarmonyOS ARM64', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD },
}

const GENERATED_PLATFORM_BY_KEY = new Map(SUPPORTED_PLATFORMS.map(p => [`${p.goos}_${p.goarch}`, p] as const))

const PLATFORMS: PlatformEntry[] = Object.entries(PLATFORM_PRESENTATION).map(([key, presentation]) => {
  const platform = GENERATED_PLATFORM_BY_KEY.get(key as SupportedPlatformKey)
  if (!platform) throw new Error(`missing generated platform ${key}`)
  return { ...platform, ...presentation }
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

export const ALL_BINARIES: BinaryInfo[] = PLATFORMS.map(toBinaryInfo)

export interface DetectedDownload {
  kind: 'detected'
  binary: BinaryInfo
  alternatives: BinaryInfo[]
}

export interface ArchitectureDownload {
  kind: 'choose-architecture'
  goos: 'darwin' | 'openharmony'
  binaries: BinaryInfo[]
  alternatives: BinaryInfo[]
}

export interface ManualDownload {
  kind: 'manual'
  binaries: BinaryInfo[]
}

export type DownloadSelection = DetectedDownload | ArchitectureDownload | ManualDownload

export const MANUAL_DOWNLOAD: ManualDownload = { kind: 'manual', binaries: ALL_BINARIES }

function architectureDownload(goos: ArchitectureDownload['goos']): ArchitectureDownload {
  return {
    kind: 'choose-architecture',
    goos,
    binaries: ALL_BINARIES.filter(binary => binary.goos === goos),
    alternatives: ALL_BINARIES.filter(binary => binary.goos !== goos),
  }
}

export const MAC_DOWNLOAD = architectureDownload('darwin')
export const HARMONY_DOWNLOAD = architectureDownload('openharmony')

// Stable selections keep detection and rendering on the same binary objects.
const INSTALLATIONS = PLATFORMS.map((platform, index) => ({
  ...platform,
  download: {
    kind: 'detected',
    binary: ALL_BINARIES[index],
    alternatives: ALL_BINARIES.filter((_, otherIndex) => otherIndex !== index),
  } satisfies DetectedDownload,
}))

export function findInstallation(goos: BinaryInfo['goos'], goarch: BinaryInfo['goarch']) {
  return INSTALLATIONS.find(platform => platform.goos === goos && platform.goarch === goarch)
}

export const MAC_COMMAND: CommandInfo = { label: 'macOS', hint: SH_HINT, command: SH_CMD, mirrorCommand: SH_MIRROR_CMD }
export const HARMONY_COMMAND: CommandInfo = {
  label: 'HarmonyOS',
  hint: msg`在鸿蒙电脑或二合一设备的终端中运行：`,
  command: SH_CMD,
  mirrorCommand: SH_MIRROR_CMD,
}

const UNIX_METHOD: InstallMethod = { label: 'Linux / macOS / HarmonyOS', command: SH_CMD, mirrorCommand: SH_MIRROR_CMD }
const WINDOWS_METHOD: InstallMethod = { label: 'Windows (PowerShell)', command: PS_CMD, mirrorCommand: PS_MIRROR_CMD }

export const METHODS: InstallMethod[] = [UNIX_METHOD, WINDOWS_METHOD]

// Platforms sharing install.sh use one row, with the detected platform omitted
// from the label under "other platforms".
export function otherMethodsFor(detected: BinaryInfo['goos']): InstallMethod[] {
  if (detected === 'windows') return [UNIX_METHOD]
  const sibling: InstallMethod = {
    ...UNIX_METHOD,
    label: detected === 'linux' ? 'macOS / HarmonyOS'
      : detected === 'darwin' ? 'Linux / HarmonyOS'
      : 'Linux / macOS',
  }
  return [sibling, WINDOWS_METHOD]
}

export const SOURCE_METHOD: InstallMethod = {
  label: '从源码编译',
  command: 'go install github.com/Zxilly/cjv/cmd/cjv@latest',
  mirrorCommand: 'GOPROXY=https://goproxy.cn,direct go install github.com/Zxilly/cjv/cmd/cjv@latest',
}
