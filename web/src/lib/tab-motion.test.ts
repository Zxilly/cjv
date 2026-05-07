import { describe, expect, it } from 'vitest'
import { getInstallTabDirection, INSTALL_TABS, type InstallTab } from './tab-motion'

describe('tab-motion', () => {
  it('keeps a stable left-to-right tab order', () => {
    expect(INSTALL_TABS).toEqual(['command', 'download', 'source'])
  })

  it('returns +1 when moving rightward', () => {
    expect(getInstallTabDirection('command', 'download')).toBe(1)
    expect(getInstallTabDirection('download', 'source')).toBe(1)
  })

  it('returns -1 when moving leftward', () => {
    expect(getInstallTabDirection('source', 'download')).toBe(-1)
    expect(getInstallTabDirection('source', 'command')).toBe(-1)
  })

  it('returns 0 when the tab does not change', () => {
    const tab: InstallTab = 'command'
    expect(getInstallTabDirection(tab, tab)).toBe(0)
  })
})
