import '@testing-library/jest-dom/vitest'
import { afterEach, beforeEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import { activateLang } from './lib/i18n'

Object.defineProperty(navigator, 'clipboard', {
  configurable: true,
  value: { writeText: vi.fn(async () => {}), readText: vi.fn(async () => '') },
})

// Tests assert the Chinese (source) strings, so pin the active locale to zh.
beforeEach(() => {
  activateLang('zh')
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(Response.json({ country: 'US' }))
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
  vi.restoreAllMocks()
  localStorage.removeItem('cjv-lang')
})
