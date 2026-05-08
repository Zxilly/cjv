import { describe, expect, it } from 'vitest'
import { computePlatformResult, usePlatform, type PlatformResult } from './use-platform'

function asReady(r: PlatformResult) {
  if (r.state !== 'ready') throw new Error(`expected ready, got ${r.state}`)
  return r
}

describe('computePlatformResult', () => {
  it('detects Windows x86_64 as ready', () => {
    const r = asReady(computePlatformResult('Windows', 'amd64'))
    expect(r.info.label).toBe('Windows x86_64')
    expect(r.info.command).toMatch(/install\.ps1/)
    expect(r.info.mirrorCommand).toMatch(/CJV_MIRROR/)
    expect(r.binary.binaryName).toBe('cjv-init.exe')
    expect(r.binary.goos).toBe('windows')
    expect(r.binary.officialUrl).toBe('/dl/official/windows_amd64/cjv-init.exe')
    expect(r.binary.mirrorUrl).toBe('/dl/mirror/windows_amd64/cjv-init.exe')
  })

  it('detects macOS ARM64 as ready', () => {
    const r = asReady(computePlatformResult('macOS', 'arm64'))
    expect(r.info.label).toBe('macOS ARM64')
    expect(r.info.command).toMatch(/install\.sh/)
    expect(r.binary.goos).toBe('darwin')
    expect(r.binary.goarch).toBe('arm64')
    expect(r.binary.binaryName).toBe('cjv-init')
  })

  it('detects macOS x86_64 with the SDK warning', () => {
    const r = asReady(computePlatformResult('Mac OS', 'amd64'))
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

  it('detects Linux ARM64', () => {
    const r = asReady(computePlatformResult('Linux', 'arm64'))
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
    const r = asReady(computePlatformResult('Windows', 'amd64'))
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
    const r = asReady(computePlatformResult('Linux', 'arm64'))
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

  it('drops the current platform from otherMethods when ready', () => {
    const win = computePlatformResult('Windows', 'amd64')
    expect(win.otherMethods.map(m => m.label)).toEqual(['Linux / macOS'])

    const mac = computePlatformResult('macOS', 'arm64')
    expect(mac.otherMethods.map(m => m.label)).toEqual(['Windows (PowerShell)'])
  })

  it('keeps every method in otherMethods when state is not ready', () => {
    const r = computePlatformResult('iOS', 'arm64')
    expect(r.otherMethods).toEqual(r.methods)
  })
})

describe('usePlatform', () => {
  it('exposes the same shape as computePlatformResult', () => {
    const r = usePlatform()
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
})
