import type { ReactElement, ReactNode } from 'react'
import { render as rtlRender, type RenderOptions } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { createI18n, type Lang } from '@/lib/i18n'
import { LanguageProvider } from '@/lib/language-provider'

export const i18n = createI18n('zh')
export function activateLang(lang: Lang) { i18n.activate(lang) }

export function Wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter initialEntries={[i18n.locale === 'en' ? '/en/' : '/']}><LanguageProvider>{children}</LanguageProvider></MemoryRouter>
}

// render wraps the UI in the Lingui provider so components using <Trans>/useLingui
// work in isolation. The active locale is pinned in test-setup.ts.
export function render(ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  return rtlRender(ui, { wrapper: Wrapper, ...options })
}

export * from '@testing-library/react'
