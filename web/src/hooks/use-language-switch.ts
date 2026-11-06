import { useEffect, useRef } from 'react'
import { useAnimationControls, useReducedMotion, type Transition } from 'framer-motion'
import { useLingui } from '@lingui/react/macro'
import { i18n, activateLang, persistLang, type Lang } from '@/lib/i18n'

// Fade the whole page through opacity 0 so the locale swap lands while invisible —
// i18n is context-driven, so any copy left mounted would otherwise flip to the new
// language instantly and defeat a crossfade.
const PAGE_FADE_OUT: Transition = { duration: 0.16, ease: 'easeIn' }
const PAGE_FADE_IN: Transition = { duration: 0.22, ease: 'easeOut' }

/**
 * Owns the language-switch animation: drives a whole-page opacity fade and swaps the
 * active locale at the invisible midpoint. Returns the controls to bind to the page
 * container and the handler to wire onto the switcher buttons.
 */
export function useLanguageSwitch() {
  const { i18n: active } = useLingui()
  const fadeControls = useAnimationControls()
  const prefersReducedMotion = useReducedMotion()
  const generation = useRef(0)
  const mounted = useRef(true)

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
      generation.current++
    }
  }, [])

  // Keep the document language attribute in sync for a11y and correct font selection.
  useEffect(() => {
    document.documentElement.lang = active.locale === 'en' ? 'en' : 'zh-CN'
  }, [active.locale])

  // Every choice invalidates pending work, including choosing the current locale.
  async function switchLang(next: Lang) {
    if (!mounted.current) return
    const request = ++generation.current
    if (next === i18n.locale || prefersReducedMotion) {
      fadeControls.stop()
      fadeControls.set({ opacity: 1 })
      if (next !== i18n.locale) {
        activateLang(next)
        persistLang(next)
      }
      return
    }
    await fadeControls.start({ opacity: 0, transition: PAGE_FADE_OUT })
    if (!mounted.current || request !== generation.current) return
    // Always fade back in, even if the locale swap throws — otherwise the page would be
    // stranded invisible at opacity 0 with no recovery path.
    try {
      activateLang(next)
      persistLang(next)
    } finally {
      if (mounted.current && request === generation.current) {
        await fadeControls.start({ opacity: 1, transition: PAGE_FADE_IN })
      }
    }
  }

  return { fadeControls, switchLang }
}
