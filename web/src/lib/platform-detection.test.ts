import { describe, expect, it } from 'vitest'
import { i18n } from '@/lib/i18n'
import { HARMONY_SAMPLES } from '../test-fixtures/harmony'
import {
  computeBrowserPlatformResult,
  computePlatformResult,
  detectBrowserPlatformResult,
  type PlatformResult,
} from '@/lib/platform-detection'
import { ALL_BINARIES } from '@/lib/platform-catalog'

const MAC_DESKTOP_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15'
const WINDOWS_ARM64_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; ARM64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
// HarmonyOS NEXT drops the 'Android' token ua-parser-modern needs to name the OS,
// so without a raw-UA fallback it would parse as undefined/Linux.
const HARMONYOS_NEXT_UA =
  'Mozilla/5.0 (Phone; OpenHarmony 5.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 ArkWeb/4.1.6.1 Mobile HuaweiBrowser/5.0.0.300'
const HARMONYOS_LEGACY_UA =
  'Mozilla/5.0 (Linux; Android 12; ALN-AL00; HMSCore 6.13.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 HarmonyOS Mobile Safari/537.36'

function asReady(r: PlatformResult) {
  if (r.state !== 'ready') throw new Error(`expected ready, got ${r.state}`)
  return r
}

function asReadyWithBinary(r: PlatformResult) {
  const ready = asReady(r)
  if (ready.download.kind !== 'detected') throw new Error('expected detected binary')
  return { ...ready, download: ready.download }
}

describe('computePlatformResult', () => {
  it('detects Windows x86_64 as ready', () => {
    const r = asReadyWithBinary(computePlatformResult('Windows', 'amd64'))
    expect(r.info.label).toBe('Windows x86_64')
    expect(r.info.command).toMatch(/install\.ps1/)
    expect(r.info.mirrorCommand).toMatch(/CJV_MIRROR/)
    expect(r.download.binary.binaryName).toBe('cjv-init.exe')
    expect(r.download.binary.goos).toBe('windows')
    expect(r.download.binary.officialUrl).toBe('/dl/official/windows_amd64/cjv-init.exe')
    expect(r.download.binary.mirrorUrl).toBe('/dl/mirror/windows_amd64/cjv-init.exe')
  })

  it('detects macOS ARM64 as ready', () => {
    const r = asReadyWithBinary(computePlatformResult('macOS', 'arm64'))
    expect(r.info.label).toBe('macOS ARM64')
    expect(r.info.command).toMatch(/install\.sh/)
    expect(r.download.binary.goos).toBe('darwin')
    expect(r.download.binary.goarch).toBe('arm64')
    expect(r.download.binary.binaryName).toBe('cjv-init')
  })

  it('detects macOS x86_64 with the SDK warning', () => {
    const r = asReadyWithBinary(computePlatformResult('Mac OS', 'amd64'))
    expect(r.info.label).toBe('macOS x86_64')
    expect(i18n._(r.info.warning!)).toMatch(/x86_64/)
    expect(i18n._(r.download.binary.warning!)).toMatch(/x86_64/)
  })

  it('marks unsupported CPU architectures on known desktop OSes as unsupported', () => {
    for (const [os, arch] of [['Linux', 'mips64'], ['Linux', 'ia32'], ['Windows', 'arm64']] as const) {
      const r = computePlatformResult(os, arch)
      expect(r.state).toBe('unsupported')
      expect(r.info.label).toBe(`${os} ${arch}`)
      expect(r.download.kind).toBe('manual')
    }
  })

  it('keeps macOS command install ready when the desktop browser hides the CPU architecture', () => {
    const r = asReady(computePlatformResult('Mac OS', ''))
    expect(r.info.label).toBe('macOS')
    expect(r.info.command).toMatch(/install\.sh/)
    expect(r.download).toMatchObject({ kind: 'choose-architecture', goos: 'darwin' })
  })

  it('detects Linux ARM64', () => {
    const r = asReadyWithBinary(computePlatformResult('Linux', 'arm64'))
    expect(r.info.label).toBe('Linux ARM64')
    expect(r.download.binary.goarch).toBe('arm64')
  })

  it('marks iOS / Android / HarmonyOS as unsupported', () => {
    for (const os of ['iOS', 'Android', 'HarmonyOS']) {
      const r = computePlatformResult(os, 'arm64')
      expect(r.state).toBe('unsupported')
      expect(r.info.label).toBe(os)
      expect(r.download.kind).toBe('manual')
    }
  })

  it('marks an empty OS as unknown', () => {
    const r = computePlatformResult('', 'amd64')
    expect(r.state).toBe('unknown')
    expect(r.info.label).toBe('未知平台')
    expect(r.download.kind).toBe('manual')
  })

  it('marks an arbitrary unrecognized OS as unknown', () => {
    const r = computePlatformResult('FreeBSD', 'amd64')
    expect(r.state).toBe('unknown')
    expect(r.info.label).toBe('未知平台')
  })

  it('returns binary refs that are reused from the platform catalog', () => {
    const r = asReadyWithBinary(computePlatformResult('Windows', 'amd64'))
    expect(ALL_BINARIES).toContain(r.download.binary)
  })

  it('lists all seven build targets in display order', () => {
    const keys = ALL_BINARIES.map(b => `${b.goos}_${b.goarch}`)
    expect(keys).toEqual([
      'windows_amd64',
      'darwin_arm64',
      'darwin_amd64',
      'linux_amd64',
      'linux_arm64',
      'openharmony_amd64',
      'openharmony_arm64',
    ])
  })

  it('produces release URLs that follow the goreleaser scheme', () => {
    const r = asReadyWithBinary(computePlatformResult('Linux', 'arm64'))
    expect(r.download.binary.officialReleaseUrl).toMatch(/cjv_linux_arm64\.tar\.gz$/)
    expect(r.download.binary.mirrorReleaseUrl).toMatch(/cjv-mirror_linux_arm64\.tar\.gz$/)
    const win = ALL_BINARIES.find(b => b.goos === 'windows')
    expect(win?.officialReleaseUrl).toMatch(/cjv_windows_amd64\.zip$/)
  })

  it('exposes the source-build method separately from the install methods table', () => {
    const r = computePlatformResult('Windows', 'amd64')
    expect(r.sourceMethod.command).toMatch(/^go install /)
    expect(r.methods.map(m => m.label)).not.toContain(r.sourceMethod.label)
  })

  it('offers a GOPROXY mirror variant for the source-build method', () => {
    const r = computePlatformResult('Windows', 'amd64')
    expect(r.sourceMethod.mirrorCommand).toBe(
      'GOPROXY=https://goproxy.cn,direct go install github.com/Zxilly/cjv/cmd/cjv@latest',
    )
  })

  it('tags mobile OSes with reason "mobile" and unsupported desktop archs with reason "arch"', () => {
    expect(computePlatformResult('iOS', 'arm64').info.reason).toBe('mobile')
    expect(computePlatformResult('Android', 'arm64').info.reason).toBe('mobile')
    expect(computePlatformResult('HarmonyOS', 'arm64').info.reason).toBe('harmony')
    expect(computePlatformResult('Windows', 'arm64').info.reason).toBe('arch')
    expect(computePlatformResult('Linux', 'mips64').info.reason).toBe('arch')
    // Unknown and ready states do not carry an unsupported reason.
    expect(computePlatformResult('FreeBSD', 'amd64').info.reason).toBeUndefined()
    expect(computePlatformResult('Windows', 'amd64').info.reason).toBeUndefined()
  })

  it('groups other platforms sharing the shell installer in one row', () => {
    // A Windows visitor sees one shared shell command for all three platforms.
    const win = computePlatformResult('Windows', 'amd64')
    expect(win.otherMethods.map(m => m.label)).toEqual(['Linux / macOS / HarmonyOS'])

    // Omit the detected platform from the shared command's label.
    const mac = computePlatformResult('macOS', 'arm64')
    expect(mac.otherMethods.map(m => m.label)).toEqual(['Linux / HarmonyOS', 'Windows (PowerShell)'])

    const macUnknownArch = computePlatformResult('macOS', '')
    expect(macUnknownArch.otherMethods.map(m => m.label)).toEqual(['Linux / HarmonyOS', 'Windows (PowerShell)'])

    const linux = computePlatformResult('Linux', 'arm64')
    expect(linux.otherMethods.map(m => m.label)).toEqual(['macOS / HarmonyOS', 'Windows (PowerShell)'])
  })

  it('keeps every method in otherMethods when state is not ready', () => {
    const r = computePlatformResult('iOS', 'arm64')
    expect(r.otherMethods).toEqual(r.methods)
  })
})

describe('computeBrowserPlatformResult', () => {
  it('keeps macOS command install ready when the browser hides the CPU architecture', () => {
    const r = asReady(computeBrowserPlatformResult({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
    }))

    expect(r.info.label).toBe('macOS')
    expect(r.download).toMatchObject({ kind: 'choose-architecture', goos: 'darwin' })
  })

  it('marks iPadOS desktop mode as unsupported instead of macOS', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 5,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('iOS')
    expect(r.download.kind).toBe('manual')
  })

  it('detects Windows ARM64 UA tokens as unsupported Windows arm64', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 0,
      platform: 'Win32',
      userAgent: WINDOWS_ARM64_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('Windows arm64')
    expect(r.info.reason).toBe('arch')
    expect(r.download.kind).toBe('manual')
  })

  it('retains the OpenHarmony version for an older NEXT UA', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 5,
      platform: 'Linux aarch64',
      userAgent: HARMONYOS_NEXT_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('HarmonyOS (OpenHarmony 5.0)')
    expect(r.info.reason).toBe('harmony')
    expect(r.download.kind).toBe('manual')
  })

  it('keeps legacy HarmonyOS without inventing an OS version from browser tokens', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 5,
      platform: 'Linux armv8l',
      userAgent: HARMONYOS_LEGACY_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('HarmonyOS')
    expect(r.info.reason).toBe('harmony')
    if (r.state !== 'unsupported') throw new Error('expected unsupported')
    expect(r.info.harmony?.version).toBeUndefined()
  })

  it.each(HARMONY_SAMPLES)('detects $name despite conflicting platform hints', async sample => {
    const input = {
      userAgent: sample.userAgent,
      platform: 'Win32',
      userAgentData: {
        platform: 'Windows',
        getHighEntropyValues: async () => ({ platform: 'Windows', architecture: 'x86', bitness: '64' }),
      },
    }
    for (const r of [computeBrowserPlatformResult(input), await detectBrowserPlatformResult(input)]) {
      expect(r.state).toBe(sample.device === 'convertible-tablet' ? 'ready' : 'unsupported')
      expect(r.info.label).toBe(`HarmonyOS (OpenHarmony ${sample.version})`)
      expect(r.info.reason).toBe(sample.device === 'convertible-tablet' ? undefined : 'harmony')
      expect(r.download.kind).toBe(sample.device === 'convertible-tablet' ? 'choose-architecture' : 'manual')
      expect(r.info.harmony).toEqual({ family: 'OpenHarmony', version: sample.version, device: sample.device })
    }
  })

  it.each([
    ['Mozilla/5.0 (Tablet; openharmony 8.2.1)', 'OpenHarmony', '8.2.1', 'tablet'],
    ['Mozilla/5.0 (HarmonyOS/4.0)', 'HarmonyOS', '4.0', 'unknown'],
    ['Mozilla/5.0 (OpenHarmony) ArkWeb/7.0 HuaweiBrowser/6.1', 'OpenHarmony', undefined, 'unknown'],
  ])('preserves explicit OS metadata for %s', (userAgent, family, version, device) => {
    // Even a MacIntel/touch compatibility hint must not override an explicit OS.
    const r = computeBrowserPlatformResult({ userAgent, platform: 'MacIntel', maxTouchPoints: 5 })
    if (r.state !== 'unsupported') throw new Error('expected unsupported')
    expect(r.info.harmony).toEqual({ family, version, device })
    expect(r.download.kind).toBe('manual')
  })

  it.each(['PC', 'Windows NT 10.0; Win64; x64'])('offers native choices for a synthetic desktop UA with %s', token => {
    const r = computeBrowserPlatformResult({
      userAgent: `Mozilla/5.0 (${token}; OpenHarmony 7.0)`,
      userAgentData: { platform: 'Windows', architecture: 'x86', bitness: '64' },
    })
    expect(r.state).toBe('ready')
    if (r.state !== 'ready') throw new Error('expected ready')
    expect(r.download).toMatchObject({ kind: 'choose-architecture', goos: 'openharmony' })
    expect(r.info.harmony?.device).toBe('pc')
  })

  it('does not identify HarmonyOS from HuaweiBrowser or ArkWeb alone', () => {
    const r = computeBrowserPlatformResult({
      userAgent: 'Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 Chrome/144.0.0.0 Mobile Safari/537.36 ArkWeb/7.0 HuaweiBrowser/6.1',
    })
    expect(r.info.label).toBe('Android')
    expect(r.info.reason).toBe('mobile')
  })


  it('uses UA Client Hints to detect macOS ARM64 when Chromium exposes them', async () => {
    const r = await detectBrowserPlatformResult({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: {
        platform: 'macOS',
        getHighEntropyValues: async hints => {
          expect(hints).toEqual(['architecture', 'bitness', 'platform'])
          return { architecture: 'arm', bitness: '64', platform: 'macOS' }
        },
      },
    })

    const ready = asReadyWithBinary(r)
    expect(ready.info.label).toBe('macOS ARM64')
    expect(ready.download.binary.goos).toBe('darwin')
    expect(ready.download.binary.goarch).toBe('arm64')
  })

  it('uses UA Client Hints to detect macOS x86_64 when Chromium exposes them', async () => {
    const r = await detectBrowserPlatformResult({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: {
        platform: 'macOS',
        getHighEntropyValues: async () => ({ architecture: 'x86', bitness: '64', platform: 'macOS' }),
      },
    })

    const ready = asReadyWithBinary(r)
    expect(ready.info.label).toBe('macOS x86_64')
    expect(i18n._(ready.info.warning!)).toMatch(/x86_64/)
  })
})
