import { afterEach, describe, expect, it } from 'vitest'
import { activateLang, detectLang, i18n, persistLang } from './i18n'

const STORAGE_KEY = 'cjv-lang'

describe('i18n', () => {
  afterEach(() => {
    localStorage.removeItem(STORAGE_KEY)
    activateLang('zh')
  })

  it('detectLang prefers a saved language', () => {
    localStorage.setItem(STORAGE_KEY, 'en')
    expect(detectLang()).toBe('en')
    localStorage.setItem(STORAGE_KEY, 'zh')
    expect(detectLang()).toBe('zh')
  })

  it('detectLang falls back to the navigator language', () => {
    localStorage.removeItem(STORAGE_KEY)
    const original = navigator.language
    try {
      Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
      expect(detectLang()).toBe('zh')
      Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true })
      expect(detectLang()).toBe('en')
    } finally {
      Object.defineProperty(navigator, 'language', { value: original, configurable: true })
    }
  })

  it('activateLang switches the active locale', () => {
    activateLang('en')
    expect(i18n.locale).toBe('en')
    activateLang('zh')
    expect(i18n.locale).toBe('zh')
  })

  it('persistLang stores the choice', () => {
    persistLang('en')
    expect(localStorage.getItem(STORAGE_KEY)).toBe('en')
  })
})
