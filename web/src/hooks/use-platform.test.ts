import { describe, expect, it } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import {
  computeBrowserPlatformResult,
  computePlatformResult,
  detectBrowserPlatformResult,
  usePlatform,
  type PlatformResult,
} from './use-platform'

const MAC_DESKTOP_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15'
const WINDOWS_DESKTOP_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
const WINDOWS_ARM64_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; ARM64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'

function asReady(r: PlatformResult) {
  if (r.state !== 'ready') throw new Error(`expected ready, got ${r.state}`)
  return r
}

function asReadyWithBinary(r: PlatformResult) {
  const ready = asReady(r)
  if (!ready.binary) throw new Error('expected ready result with binary')
  return ready as typeof ready & { binary: NonNullable<typeof ready.binary> }
}

describe('computePlatformResult', () => {
  it('detects Windows x86_64 as ready', () => {
    const r = asReadyWithBinary(computePlatformResult('Windows', 'amd64'))
    expect(r.info.label).toBe('Windows x86_64')
    expect(r.info.command).toMatch(/install\.ps1/)
    expect(r.info.mirrorCommand).toMatch(/CJV_MIRROR/)
    expect(r.binary.binaryName).toBe('cjv-init.exe')
    expect(r.binary.goos).toBe('windows')
    expect(r.binary.officialUrl).toBe('/dl/official/windows_amd64/cjv-init.exe')
    expect(r.binary.mirrorUrl).toBe('/dl/mirror/windows_amd64/cjv-init.exe')
  })

  it('detects macOS ARM64 as ready', () => {
    const r = asReadyWithBinary(computePlatformResult('macOS', 'arm64'))
    expect(r.info.label).toBe('macOS ARM64')
    expect(r.info.command).toMatch(/install\.sh/)
    expect(r.binary.goos).toBe('darwin')
    expect(r.binary.goarch).toBe('arm64')
    expect(r.binary.binaryName).toBe('cjv-init')
  })

  it('detects macOS x86_64 with the SDK warning', () => {
    const r = asReadyWithBinary(computePlatformResult('Mac OS', 'amd64'))
    expect(r.info.label).toBe('macOS x86_64')
    expect(r.info.warning).toMatch(/x86_64/)
    expect(r.binary.warning).toMatch(/x86_64/)
  })

  it('marks unsupported CPU architectures on known desktop OSes as unsupported', () => {
    for (const [os, arch] of [['Linux', 'mips64'], ['Linux', 'ia32'], ['Windows', 'arm64']] as const) {
      const r = computePlatformResult(os, arch)
      expect(r.state).toBe('unsupported')
      expect(r.info.label).toBe(`${os} ${arch}`)
      expect(r.binary).toBeNull()
    }
  })

  it('keeps macOS command install ready when the desktop browser hides the CPU architecture', () => {
    const r = asReady(computePlatformResult('Mac OS', ''))
    expect(r.info.label).toBe('macOS')
    expect(r.info.command).toMatch(/install\.sh/)
    expect(r.binary).toBeNull()
  })

  it('detects Linux ARM64', () => {
    const r = asReadyWithBinary(computePlatformResult('Linux', 'arm64'))
    expect(r.info.label).toBe('Linux ARM64')
    expect(r.binary.goarch).toBe('arm64')
  })

  it('marks iOS / Android / HarmonyOS as unsupported', () => {
    for (const os of ['iOS', 'Android', 'HarmonyOS']) {
      const r = computePlatformResult(os, 'arm64')
      expect(r.state).toBe('unsupported')
      expect(r.info.label).toBe(os)
      expect(r.binary).toBeNull()
    }
  })

  it('marks an empty OS as unknown', () => {
    const r = computePlatformResult('', 'amd64')
    expect(r.state).toBe('unknown')
    expect(r.info.label).toBe('未知平台')
    expect(r.binary).toBeNull()
  })

  it('marks an arbitrary unrecognized OS as unknown', () => {
    const r = computePlatformResult('FreeBSD', 'amd64')
    expect(r.state).toBe('unknown')
    expect(r.info.label).toBe('未知平台')
  })

  it('returns binary refs that are reused inside allBinaries', () => {
    const r = asReadyWithBinary(computePlatformResult('Windows', 'amd64'))
    expect(r.allBinaries).toContain(r.binary)
  })

  it('lists all five build targets under allBinaries', () => {
    const r = computePlatformResult('Linux', 'amd64')
    const keys = r.allBinaries.map(b => `${b.goos}_${b.goarch}`)
    expect(keys).toEqual([
      'windows_amd64',
      'darwin_arm64',
      'darwin_amd64',
      'linux_amd64',
      'linux_arm64',
    ])
  })

  it('produces release URLs that follow the goreleaser scheme', () => {
    const r = asReadyWithBinary(computePlatformResult('Linux', 'arm64'))
    expect(r.binary.officialReleaseUrl).toMatch(/cjv_linux_arm64\.tar\.gz$/)
    expect(r.binary.mirrorReleaseUrl).toMatch(/cjv-mirror_linux_arm64\.tar\.gz$/)
    const win = r.allBinaries.find(b => b.goos === 'windows')
    expect(win?.officialReleaseUrl).toMatch(/cjv_windows_amd64\.zip$/)
  })

  it('exposes the source-build method separately from the install methods table', () => {
    const r = computePlatformResult('Windows', 'amd64')
    expect(r.sourceMethod.command).toMatch(/^go install /)
    expect(r.methods.map(m => m.label)).not.toContain(r.sourceMethod.label)
  })

  it('lists the other platforms in otherMethods, naming the Unix sibling for Linux/macOS visitors', () => {
    // A non-Unix visitor keeps the combined "Linux / macOS" row (one shared command).
    const win = computePlatformResult('Windows', 'amd64')
    expect(win.otherMethods.map(m => m.label)).toEqual(['Linux / macOS'])

    // A macOS visitor still needs Linux listed, named explicitly so it is not dropped
    // along with their own platform's combined row.
    const mac = computePlatformResult('macOS', 'arm64')
    expect(mac.otherMethods.map(m => m.label)).toEqual(['Linux', 'Windows (PowerShell)'])

    const macUnknownArch = computePlatformResult('macOS', '')
    expect(macUnknownArch.otherMethods.map(m => m.label)).toEqual(['Linux', 'Windows (PowerShell)'])

    // Symmetric: a Linux visitor sees macOS named as the sibling.
    const linux = computePlatformResult('Linux', 'arm64')
    expect(linux.otherMethods.map(m => m.label)).toEqual(['macOS', 'Windows (PowerShell)'])
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
    expect(r.binary).toBeNull()
  })

  it('marks iPadOS desktop mode as unsupported instead of macOS', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 5,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('iOS')
    expect(r.binary).toBeNull()
  })

  it('detects Windows ARM64 UA tokens as unsupported Windows arm64', () => {
    const r = computeBrowserPlatformResult({
      maxTouchPoints: 0,
      platform: 'Win32',
      userAgent: WINDOWS_ARM64_UA,
    })

    expect(r.state).toBe('unsupported')
    expect(r.info.label).toBe('Windows arm64')
    expect(r.binary).toBeNull()
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
    expect(ready.binary.goos).toBe('darwin')
    expect(ready.binary.goarch).toBe('arm64')
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
    expect(ready.info.warning).toMatch(/x86_64/)
  })
})

describe('usePlatform', () => {
  it('exposes the same shape as computePlatformResult', () => {
    const input = {
      maxTouchPoints: 0,
      platform: 'Win32',
      userAgent: WINDOWS_DESKTOP_UA,
    }
    const { result } = renderHook(() => usePlatform(input))

    const r = result.current
    expect(r).toMatchObject({
      state: expect.any(String),
      info: expect.any(Object),
      methods: expect.any(Array),
      otherMethods: expect.any(Array),
      sourceMethod: expect.any(Object),
      allBinaries: expect.any(Array),
    })
    expect(r.allBinaries).toHaveLength(5)
  })

  it('updates when asynchronous UA Client Hints refine a macOS architecture', async () => {
    const input = {
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: {
        platform: 'macOS',
        getHighEntropyValues: async () => ({ architecture: 'arm', bitness: '64', platform: 'macOS' }),
      },
    }
    const { result } = renderHook(() => usePlatform(input))

    expect(result.current.state).toBe('ready')
    expect(result.current.info.label).toBe('macOS')
    expect(result.current.binary).toBeNull()
    await waitFor(() => expect(result.current.info.label).toBe('macOS ARM64'))
    expect(result.current.state).toBe('ready')
    expect(result.current.binary?.goarch).toBe('arm64')
  })
})
