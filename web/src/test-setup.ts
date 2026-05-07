import '@testing-library/jest-dom/vitest'
import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

Object.defineProperty(navigator, 'clipboard', {
  configurable: true,
  value: { writeText: vi.fn(async () => {}), readText: vi.fn(async () => '') },
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})
