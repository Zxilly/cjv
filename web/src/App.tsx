import { useState, type ReactElement } from 'react'
import { AnimatePresence, LayoutGroup, motion, useReducedMotion, type Transition, type Variants } from 'framer-motion'
import { ChevronRight } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Switch } from '@/components/ui/switch'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { CodeBlock } from '@/components/code-block'
import { CollapsibleSection } from '@/components/collapsible-section'
import { BinaryInstall } from '@/components/binary-install'
import { usePlatform, type InstallMethod } from '@/hooks/use-platform'
import { getInstallTabDirection, type InstallTab } from '@/lib/tab-motion'

function MethodsList({ items, mirror }: { items: InstallMethod[]; mirror: boolean }) {
  return (
    <div className="space-y-3 text-sm">
      {items.map(m => (
        <div key={m.label}>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">{m.label}</p>
          <CodeBlock command={mirror && m.mirrorCommand ? m.mirrorCommand : m.command} />
        </div>
      ))}
    </div>
  )
}

const TAB_CONTENT_CLS =
  'w-full text-base divide-y divide-gray-200 dark:divide-gray-800'

const TAB_TRIGGER_CLS =
  'flex-1 h-full rounded-none data-active:text-cj dark:data-active:text-cj-light data-active:after:bg-cj dark:data-active:after:bg-cj-light data-active:after:opacity-100 after:bottom-[-1px]'

const panelVariants: Variants = {
  enter: (direction: number) => ({ x: direction >= 0 ? '100%' : '-100%' }),
  center: { x: 0 },
  exit: (direction: number) => ({ x: direction >= 0 ? '-100%' : '100%' }),
}

const SLIDE_SPRING: Transition = { type: 'spring', stiffness: 420, damping: 42, mass: 0.85 }
const HEIGHT_EASE: Transition = { duration: 0.28, ease: [0.4, 0, 0.2, 1] as const }
const NO_MOTION: Transition = { duration: 0 }

const MIRROR_TAP = { scale: 0.995 }
const MIRROR_REST = { scale: 1 }
const MIRROR_ACTIVE = { scale: 1.04 }
const LABEL_INITIAL = { opacity: 0, y: 4 }
const LABEL_ANIMATE = { opacity: 1, y: 0 }
const LABEL_EXIT = { opacity: 0, y: -4 }

export default function App() {
  const platform = usePlatform()
  const { methods, otherMethods, sourceMethod, allBinaries } = platform
  const prefersReducedMotion = useReducedMotion()
  const [tab, setTab] = useState<InstallTab>('command')
  const [slideDirection, setSlideDirection] = useState(0)
  const [mirror, setMirror] = useState(false)
  const [otherMethodsOpen, setOtherMethodsOpen] = useState(false)

  const slideTransition = prefersReducedMotion ? NO_MOTION : SLIDE_SPRING
  const heightTransition = prefersReducedMotion ? NO_MOTION : HEIGHT_EASE
  // macOS on Safari/Firefox exposes no CPU architecture, so detection lands in the
  // 'ready' state with no concrete binary. Only then do we offer an explicit Apple
  // Silicon / Intel choice; once the arch is known (Chromium via UA Client Hints) the
  // detected single binary is shown instead. This is the sole ready+null-binary case.
  const showMacOSDownloadChoices = platform.state === 'ready' && platform.binary === null

  function handleTabChange(value: string) {
    const next = value as InstallTab

    setSlideDirection(getInstallTabDirection(tab, next))
    setTab(next)
  }

  const panels: Record<InstallTab, ReactElement> = {
    command: (
      <TabsContent forceMount value="command" className={TAB_CONTENT_CLS}>
        {platform.state === 'ready' && (
          <>
            <div className="p-6 text-center">
              <p className="text-base text-gray-500 dark:text-gray-400 mb-4">{platform.info.hint}</p>
              <CodeBlock command={mirror ? platform.info.mirrorCommand : platform.info.command} primary />
              <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">检测到你的平台：{platform.info.label}</p>
              {platform.info.warning && (
                <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">⚠ {platform.info.warning}</p>
              )}
            </div>
            {otherMethods.length > 0 && (
              <CollapsibleSection title="其他平台" initial={false}>
                <div className="mt-3"><MethodsList items={otherMethods} mirror={mirror} /></div>
              </CollapsibleSection>
            )}
          </>
        )}

        {platform.state === 'unknown' && (
          <>
            <div className="p-6 text-center">
              <p className="text-base text-gray-500 dark:text-gray-400">无法识别你的平台，以下是所有支持的安装方式。</p>
            </div>
            <div className="px-6 pb-6"><MethodsList items={methods} mirror={mirror} /></div>
          </>
        )}

        {platform.state === 'unsupported' && <div className="px-6 py-5"><MethodsList items={methods} mirror={mirror} /></div>}
      </TabsContent>
    ),
    download: (
      <TabsContent forceMount value="download" className={TAB_CONTENT_CLS}>
        <BinaryInstall
          binary={platform.binary}
          allBinaries={allBinaries}
          mirror={mirror}
          showMacOSChoices={showMacOSDownloadChoices}
        />
      </TabsContent>
    ),
    source: (
      <TabsContent forceMount value="source" className={TAB_CONTENT_CLS}>
        <div className="p-6 text-center">
          <p className="text-base text-gray-500 dark:text-gray-400 mb-4">使用 Go 从源码编译安装：</p>
          <CodeBlock command={sourceMethod.command} primary />
          <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">需要本机已安装 Go 环境。</p>
        </div>
      </TabsContent>
    ),
  }

  const installCard = (
    <>
      <Tabs value={tab} onValueChange={handleTabChange} className="flex-col gap-0">
        <TabsList
          variant="line"
          className="w-full h-11 p-0 border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/50 rounded-none gap-0"
        >
          <TabsTrigger value="command" className={TAB_TRIGGER_CLS}>命令安装</TabsTrigger>
          <TabsTrigger value="download" className={TAB_TRIGGER_CLS}>下载安装</TabsTrigger>
          <TabsTrigger value="source" className={TAB_TRIGGER_CLS}>编译安装</TabsTrigger>
        </TabsList>

        <div className="relative overflow-hidden">
          <AnimatePresence initial={false} custom={slideDirection} mode="popLayout">
            <motion.div
              key={tab}
              custom={slideDirection}
              variants={panelVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className="w-full"
              data-slide-direction={slideDirection}
            >
              {panels[tab]}
            </motion.div>
          </AnimatePresence>
        </div>
      </Tabs>

      <motion.label
        layout="position"
        layoutDependency={mirror}
        whileTap={prefersReducedMotion ? undefined : MIRROR_TAP}
        transition={heightTransition}
        className="px-6 py-3.5 flex items-center gap-3 cursor-pointer border-t border-gray-200 dark:border-gray-800 bg-gray-50/60 dark:bg-gray-900/30 select-none"
      >
        <motion.span
          animate={prefersReducedMotion || !mirror ? MIRROR_REST : MIRROR_ACTIVE}
          transition={heightTransition}
          className="inline-flex"
        >
          <Switch
            checked={mirror}
            onCheckedChange={setMirror}
            className="data-checked:bg-cj! dark:data-checked:bg-cj-light!"
          />
        </motion.span>
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">使用镜像源</span>
        <span className="ml-auto text-xs text-gray-400 dark:text-gray-500 overflow-hidden">
          <AnimatePresence mode="wait" initial={false}>
            <motion.span
              key={mirror ? 'mirror' : 'official'}
              initial={LABEL_INITIAL}
              animate={LABEL_ANIMATE}
              exit={LABEL_EXIT}
              transition={heightTransition}
              className="block"
            >
              {mirror ? 'GitCode · 镜像源' : 'GitHub · 默认源'}
            </motion.span>
          </AnimatePresence>
        </span>
      </motion.label>
    </>
  )

  return (
    <LayoutGroup id="install-layout">
      <div className="max-w-2xl mx-auto px-4 md:px-6 py-12 md:py-24 w-full">
        <motion.header layout="position" layoutDependency={tab} transition={heightTransition}>
          <h1 className="text-6xl sm:text-7xl md:text-8xl" style={{ fontFamily: '"Patua One", serif', fontWeight: 400 }}>
            <span className="cj-gradient" style={{ paddingBottom: '0.15em', display: 'inline-block' }}>cjv</span>
          </h1>
          <p className="mt-6 text-lg md:text-2xl text-gray-500 dark:text-gray-400">
            <a href="https://cangjie-lang.cn/" target="_blank" rel="noopener noreferrer" className="text-cj dark:text-cj-light hover:underline mr-1">仓颉</a>编程语言 SDK 工具链管理器
          </p>
        </motion.header>

        <motion.div
          layout
          layoutDependency={platform.state === 'unsupported' ? otherMethodsOpen : tab}
          className="mt-10 rounded-lg border border-gray-200 dark:border-gray-800 overflow-hidden"
          transition={heightTransition}
        >
          {platform.state === 'unsupported' ? (
            <>
              <div className="p-6 text-center">
                <p className="text-base text-gray-500 dark:text-gray-400">
                  cjv 暂不支持 <strong className="text-gray-700 dark:text-gray-300">{platform.info.label}</strong> 平台。
                </p>
                <p className="mt-2 text-sm text-gray-400 dark:text-gray-500">cjv 目前仅在 Linux、macOS 和 Windows 桌面系统上提供。</p>
                <p className="mt-3 text-sm text-gray-400 dark:text-gray-500">
                  请在桌面设备上访问此页面，或查看{' '}
                  <a href="https://github.com/Zxilly/cjv" target="_blank" rel="noopener noreferrer" className="text-cj dark:text-cj-light hover:underline">GitHub</a>{' '}
                  了解更多。
                </p>
              </div>
              <Collapsible
                open={otherMethodsOpen}
                onOpenChange={setOtherMethodsOpen}
                className="border-t border-gray-200 dark:border-gray-800"
              >
                <CollapsibleTrigger className="group cursor-pointer px-6 py-4 w-full text-left text-sm text-gray-500 dark:text-gray-400 hover:text-cj dark:hover:text-cj-light flex items-center gap-1.5">
                  <ChevronRight className="size-4 transition-transform duration-200 group-data-[state=open]:rotate-90" />
                  <span className="group-hover:underline">查看其他平台的安装方式</span>
                </CollapsibleTrigger>
                <CollapsibleContent className="overflow-hidden data-[state=open]:animate-collapsible-down data-[state=closed]:animate-collapsible-up">
                  <div className="border-t border-gray-200 dark:border-gray-800">{installCard}</div>
                </CollapsibleContent>
              </Collapsible>
            </>
          ) : (
            installCard
          )}
        </motion.div>

        <motion.nav layout="position" layoutDependency={tab} transition={heightTransition} className="mt-12 text-center text-base text-gray-400 dark:text-gray-600 space-x-3">
          <a href="https://github.com/Zxilly/cjv" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light">GitHub</a>
          <span>·</span>
          <a href="https://github.com/Zxilly/cjv/releases" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light">Releases</a>
          <span>·</span>
          <a href="https://cangjie-lang.cn/" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light">仓颉官网</a>
        </motion.nav>
      </div>
    </LayoutGroup>
  )
}
