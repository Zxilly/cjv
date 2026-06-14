import { useEffect } from 'react'
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

  // Keep the document language attribute in sync for a11y and correct font selection.
  useEffect(() => {
    document.documentElement.lang = active.locale === 'en' ? 'en' : 'zh-CN'
  }, [active.locale])

  // Concurrent clicks just interrupt each other on the shared controls — last wins.
  async function switchLang(next: Lang) {
    if (next === i18n.locale) return
    if (prefersReducedMotion) {
      activateLang(next)
      persistLang(next)
      return
    }
    await fadeControls.start({ opacity: 0, transition: PAGE_FADE_OUT })
    // Always fade back in, even if the locale swap throws — otherwise the page would be
    // stranded invisible at opacity 0 with no recovery path.
    try {
      activateLang(next)
      persistLang(next)
    } finally {
      await fadeControls.start({ opacity: 1, transition: PAGE_FADE_IN })
    }
  }

  return { fadeControls, switchLang }
}
