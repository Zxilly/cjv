import { i18n } from '@lingui/core'
import { messages as zhMessages } from '@/locales/zh/messages.po'
import { messages as enMessages } from '@/locales/en/messages.po'

export type Lang = 'zh' | 'en'

export const LANGS: readonly Lang[] = ['zh', 'en']

const STORAGE_KEY = 'cjv-lang'
const catalogs: Record<Lang, typeof zhMessages> = { zh: zhMessages, en: enMessages }

export function detectLang(): Lang {
  if (typeof window === 'undefined') return 'en'
  try {
    const saved = window.localStorage?.getItem(STORAGE_KEY)
    if (saved === 'zh' || saved === 'en') return saved
  } catch {
    // Reading localStorage can throw (sandboxed iframe, blocked storage); fall
    // through to the navigator language rather than aborting module evaluation.
  }
  if (/^zh\b/i.test(window.navigator?.language ?? '')) return 'zh'
  return 'en'
}

export function activateLang(lang: Lang): void {
  i18n.load(lang, catalogs[lang])
  i18n.activate(lang)
}

export function persistLang(lang: Lang): void {
  try {
    window.localStorage?.setItem(STORAGE_KEY, lang)
  } catch {
    // Ignore storage failures (private mode, disabled storage, etc.).
  }
}

// Activate the detected language eagerly so the first render is already localized.
activateLang(detectLang())

export { i18n }
