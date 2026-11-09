import { setupI18n } from '@lingui/core'
import { messages as zhMessages } from '@/locales/zh/messages.po'
import { messages as enMessages } from '@/locales/en/messages.po'

export type Lang = 'zh' | 'en'
export const LANGS: readonly Lang[] = ['zh', 'en']

export function languageForPath(pathname: string): Lang {
  return pathname === '/en' || pathname.startsWith('/en/') ? 'en' : 'zh'
}

export function createI18n(lang: Lang) {
  return setupI18n({ locale: lang, messages: { zh: zhMessages, en: enMessages } })
}

export function persistLang(lang: Lang): void {
  try {
    window.localStorage?.setItem('cjv-lang', lang)
  } catch {
    // Ignore storage failures (private mode, disabled storage, etc.).
  }
}
