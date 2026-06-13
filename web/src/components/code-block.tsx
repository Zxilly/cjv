import { useEffect, useRef, useState } from 'react'
import { Check, Copy, X } from 'lucide-react'
import { motion, AnimatePresence, type Transition } from 'framer-motion'
import { useLingui } from '@lingui/react/macro'
import { cn } from '@/lib/utils'

interface CodeBlockProps {
  command: string
  primary?: boolean
}

type CopyStatus = 'idle' | 'copied' | 'error'

const COMMAND_TRANSITION: Transition = { duration: 0.18, ease: [0.4, 0, 0.2, 1] as const }
const ICON_TRANSITION: Transition = { duration: 0.15 }

// Fallback for non-secure contexts (HTTP) or browsers/permissions that block the async
// Clipboard API: drop a hidden textarea, select it, and ask execCommand to copy.
function legacyCopy(text: string): boolean {
  if (typeof document === 'undefined') return false
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '-9999px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(textarea)
  return ok
}

async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // fall through to the legacy path below
    }
  }
  return legacyCopy(text)
}

export function CodeBlock({ command, primary }: CodeBlockProps) {
  const { t } = useLingui()
  const [status, setStatus] = useState<CopyStatus>('idle')
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  const iconCls = primary ? 'w-4 h-4' : 'w-3.5 h-3.5'

  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])
  useEffect(() => {
    setStatus('idle')
    if (timer.current) {
      clearTimeout(timer.current)
      timer.current = undefined
    }
  }, [command])

  async function copy() {
    const ok = await copyToClipboard(command)
    setStatus(ok ? 'copied' : 'error')
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(() => setStatus('idle'), ok ? 1500 : 2500)
  }

  const copied = status === 'copied'
  const errored = status === 'error'

  return (
    <div
      className={cn(
        'install-box relative bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700',
        primary ? 'rounded-lg' : 'rounded',
      )}
    >
      <div className={cn('install-box-scroll overflow-x-auto overflow-y-hidden', primary ? 'px-5 py-4 pr-12' : 'px-3 py-2 pr-9')}>
        <AnimatePresence mode="wait" initial={false}>
          <motion.code
            key={command}
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={COMMAND_TRANSITION}
            className={cn(
              'block w-max font-mono text-gray-900 dark:text-gray-100 whitespace-nowrap',
              primary ? 'text-sm md:text-base' : 'text-sm',
            )}
          >
            {command}
          </motion.code>
        </AnimatePresence>
      </div>
      <button
        type="button"
        aria-label={t`复制命令`}
        title={t`复制`}
        onClick={copy}
        className={cn(
          'absolute top-1/2 -translate-y-1/2 rounded bg-gray-50/80 dark:bg-gray-900/80 backdrop-blur-sm hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors cursor-pointer text-gray-400 hover:text-gray-600 dark:hover:text-gray-300',
          primary ? 'right-3 p-1.5' : 'right-2 p-1',
        )}
      >
        <AnimatePresence mode="wait" initial={false}>
          <motion.span
            key={status}
            initial={{ scale: 0.6, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.6, opacity: 0 }}
            transition={ICON_TRANSITION}
            className={cn(
              'block',
              copied && 'text-cj dark:text-cj-light',
              errored && 'text-red-500 dark:text-red-400',
            )}
          >
            {copied
              ? <Check className={iconCls} strokeWidth={2.5} />
              : errored
                ? <X className={iconCls} strokeWidth={2.5} />
                : <Copy className={iconCls} strokeWidth={2} />}
          </motion.span>
        </AnimatePresence>
      </button>
      <span aria-live="polite" className="sr-only">
        {copied ? t`已复制` : errored ? t`复制失败，请手动复制` : ''}
      </span>
    </div>
  )
}
