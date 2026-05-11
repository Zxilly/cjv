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
  return {
    ...common,
    state: UNSUPPORTED.has(os) || knownDesktopOS ? 'unsupported' : 'unknown',
    info: { label: UNSUPPORTED.has(os) ? os : knownDesktopOS ? `${os} ${arch}` : '未知平台' },
    binary: null,
  }
}

const PLATFORM_RESULT = computePlatformResult(parseOS().name || '', parseCPU().architecture || 'amd64')

export function usePlatform() {
  return PLATFORM_RESULT
}
