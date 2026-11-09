import { useMemo, type ReactNode } from 'react'
import { I18nProvider } from '@lingui/react'
import { useLocation } from 'react-router'
import { createI18n, languageForPath } from './i18n'

export function LanguageProvider({ children }: { children: ReactNode }) {
  const { pathname } = useLocation()
  const lang = languageForPath(pathname)
  // Each render tree owns its catalogs, including concurrent server renders.
  const i18n = useMemo(() => createI18n(lang), [lang])
  return <I18nProvider i18n={i18n}>{children}</I18nProvider>
}
