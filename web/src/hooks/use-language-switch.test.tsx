import { expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { I18nProvider } from '@lingui/react'
import type { ReactNode } from 'react'
import { i18n } from '@/lib/i18n'
import { useLanguageSwitch } from './use-language-switch'

const animation = vi.hoisted(() => ({
  start: vi.fn<(target: { opacity: number }) => Promise<void>>(),
  set: vi.fn(),
  stop: vi.fn(),
}))

vi.mock('framer-motion', () => ({
  useAnimationControls: () => animation,
  useReducedMotion: () => false,
}))

function wrapper({ children }: { children: ReactNode }) {
  return <I18nProvider i18n={i18n}>{children}</I18nProvider>
}

it('the last language choice wins even when it is the currently displayed language', async () => {
  let finishFade!: () => void
  animation.start.mockImplementation(({ opacity }) => opacity === 0
    ? new Promise<void>(resolve => { finishFade = resolve })
    : Promise.resolve())
  const { result } = renderHook(() => useLanguageSwitch(), { wrapper })
  let first!: Promise<void>
  act(() => { first = result.current.switchLang('en') })
  await act(async () => { await result.current.switchLang('zh') })
  await act(async () => { finishFade(); await first })
  expect(i18n.locale).toBe('zh')
  expect(localStorage.getItem('cjv-lang')).not.toBe('en')
  expect(animation.stop).toHaveBeenCalled()
  expect(animation.set).toHaveBeenCalledWith({ opacity: 1 })
})

it('an unmounted switch cannot commit a pending language change', async () => {
  let finishFade!: () => void
  animation.start.mockImplementation(({ opacity }) => opacity === 0
    ? new Promise<void>(resolve => { finishFade = resolve })
    : Promise.resolve())
  const { result, unmount } = renderHook(() => useLanguageSwitch(), { wrapper })
  let pending!: Promise<void>
  act(() => { pending = result.current.switchLang('en') })
  unmount()
  await act(async () => { finishFade(); await pending })
  expect(i18n.locale).toBe('zh')
  expect(localStorage.getItem('cjv-lang')).not.toBe('en')
})
