import { expect, it } from 'vitest'
import { createI18n, languageForPath, persistLang } from './i18n'

it('keeps concurrent language render trees independent', () => {
  const zh = createI18n('zh')
  const en = createI18n('en')
  expect(zh.locale).toBe('zh')
  expect(en.locale).toBe('en')
  en.activate('zh')
  zh.activate('en')
  expect(en.locale).toBe('zh')
  expect(zh.locale).toBe('en')
})

it('uses the explicit route regardless of a saved language', () => {
  localStorage.setItem('cjv-lang', 'en')
  expect(languageForPath('/')).toBe('zh')
  expect(languageForPath('/en/')).toBe('en')
  expect(languageForPath('/en')).toBe('en')
})

it('stores the chosen language', () => {
  persistLang('en')
  expect(localStorage.getItem('cjv-lang')).toBe('en')
})
