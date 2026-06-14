import { Fragment } from 'react'
import { motion, type Transition } from 'framer-motion'
import { useLingui } from '@lingui/react/macro'
import { LANGS, type Lang } from '@/lib/i18n'
import { cn } from '@/lib/utils'

const LANG_LABEL: Record<Lang, string> = { zh: '中文', en: 'English' }

export function LangSwitch({ onSelect, transition, layoutDependency }: {
  onSelect: (l: Lang) => void
  transition: Transition
  layoutDependency: unknown
}) {
  // Subscribe to locale changes so the active-language highlight re-renders here,
  // rather than relying on an ancestor's re-render.
  const { i18n: active } = useLingui()
  const lang = active.locale as Lang
  return (
    // Slide in lockstep with the header when the centered layout reflows, using the
    // same layout="position" config so the corner switcher tracks the cjv title.
    <motion.div
      layout="position"
      layoutDependency={layoutDependency}
      transition={transition}
      className="absolute right-4 top-4 md:right-6 md:top-6 flex items-center gap-1.5 text-sm"
    >
      {LANGS.map((l, idx) => (
        <Fragment key={l}>
          {idx > 0 && <span className="text-gray-300 dark:text-gray-700">/</span>}
          <button
            type="button"
            onClick={() => onSelect(l)}
            aria-current={l === lang}
            className={cn(
              'transition-colors duration-300 ease-in-out motion-reduce:transition-none',
              l === lang
                ? 'text-cj dark:text-cj-light font-medium'
                : 'text-gray-400 dark:text-gray-600 hover:text-cj dark:hover:text-cj-light cursor-pointer',
            )}
          >
            {LANG_LABEL[l]}
          </button>
        </Fragment>
      ))}
    </motion.div>
  )
}
