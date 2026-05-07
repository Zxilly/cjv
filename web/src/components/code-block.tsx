import { useEffect, useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { motion, AnimatePresence, type Transition } from 'framer-motion'
import { cn } from '@/lib/utils'

interface CodeBlockProps {
  command: string
  primary?: boolean
}

const COMMAND_TRANSITION: Transition = { duration: 0.18, ease: [0.4, 0, 0.2, 1] as const }
const ICON_TRANSITION: Transition = { duration: 0.15 }

export function CodeBlock({ command, primary }: CodeBlockProps) {
  const [copied, setCopied] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  const iconCls = primary ? 'w-4 h-4' : 'w-3.5 h-3.5'

  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])

  function copy() {
    navigator.clipboard.writeText(command).then(() => {
      setCopied(true)
      if (timer.current) clearTimeout(timer.current)
      timer.current = setTimeout(() => setCopied(false), 1500)
    })
  }

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
        title="复制"
        onClick={copy}
        className={cn(
          'absolute top-1/2 -translate-y-1/2 rounded bg-gray-50/80 dark:bg-gray-900/80 backdrop-blur-sm hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors cursor-pointer text-gray-400 hover:text-gray-600 dark:hover:text-gray-300',
          primary ? 'right-3 p-1.5' : 'right-2 p-1',
        )}
      >
        <AnimatePresence mode="wait" initial={false}>
          <motion.span
            key={copied ? 'check' : 'copy'}
            initial={{ scale: 0.6, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.6, opacity: 0 }}
            transition={ICON_TRANSITION}
            className={cn('block', copied && 'text-cj dark:text-cj-light')}
          >
            {copied
              ? <Check className={iconCls} strokeWidth={2.5} />
              : <Copy className={iconCls} strokeWidth={2} />}
          </motion.span>
        </AnimatePresence>
      </button>
    </div>
  )
}
