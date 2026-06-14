import { motion, type Transition } from 'framer-motion'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export type TabItem = { value: string; label: ReactNode }

/**
 * Green active-tab overlay. Triggers are equal-width (flex-1), so the underline bar
 * positions are pure percentages — no measuring.
 *
 * Each green label is drawn once as ordinary text (crisp subpixel AA) and revealed by
 * animating the width of an overflow-hidden wrapper — never by clip-path/transform on
 * the glyphs, which would re-rasterise them every frame (the jagged, jittery look). The
 * wrapper width is a percentage of an invisible same-text spacer, so the fill is
 * proportional to the text width, not the wider tab cell / underline.
 *
 * The reveal edge follows the slide direction so the fill tracks the bar: sliding right
 * fills L→R, sliding left fills R→L (and the outgoing label empties toward the side it
 * leaves). The bar keeps the cell width and slides; everything shares one transition.
 */
export function SlidingTabIndicator({ tabs, activeValue, direction, transition }: {
  tabs: TabItem[]
  activeValue: string
  direction: number
  transition: Transition
}) {
  const n = tabs.length
  const i = Math.max(0, tabs.findIndex(t => t.value === activeValue))

  return (
    <div aria-hidden className="pointer-events-none absolute inset-0">
      <div className="flex h-full">
        {tabs.map((t, idx) => {
          const active = idx === i
          // Incoming fills from the side the bar arrives from; outgoing empties toward
          // the side the bar leaves — both flip with the slide direction.
          const fromRight = active ? direction < 0 : direction > 0
          return (
            <span
              key={t.value}
              className="flex-1 inline-flex items-center justify-center border border-transparent px-1.5 py-0.5 text-sm font-medium whitespace-nowrap"
            >
              <span className="inline-grid">
                {/* Sizes the grid cell to the text and reserves the line box. */}
                <span className="col-start-1 row-start-1 invisible">{t.label}</span>
                {/* Reveal by width (% of text), clipped by overflow — glyphs untouched.
                    Pin the text to the reveal edge so it uncovers in the right direction. */}
                <motion.span
                  className={cn(
                    'col-start-1 row-start-1 flex items-center overflow-hidden',
                    fromRight ? 'justify-self-end justify-end' : 'justify-self-start justify-start',
                  )}
                  initial={false}
                  animate={{ width: active ? '100%' : '0%' }}
                  transition={transition}
                >
                  <span className="w-max shrink-0 text-cj dark:text-cj-light">{t.label}</span>
                </motion.span>
              </span>
            </span>
          )
        })}
      </div>

      <motion.div
        className="absolute bottom-[-1px] left-0 h-0.5 bg-cj dark:bg-cj-light"
        style={{ width: `${100 / n}%` }}
        initial={false}
        animate={{ x: `${i * 100}%` }}
        transition={transition}
      />
    </div>
  )
}
