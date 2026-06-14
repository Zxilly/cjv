import { useState, type ReactElement } from 'react'
import { AnimatePresence, LayoutGroup, motion, useReducedMotion, type Transition, type Variants } from 'framer-motion'
import { ChevronRight } from 'lucide-react'
import { I18nProvider } from '@lingui/react'
import { Trans, useLingui } from '@lingui/react/macro'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Switch } from '@/components/ui/switch'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { CodeBlock } from '@/components/code-block'
import { CollapsibleSection } from '@/components/collapsible-section'
import { BinaryInstall } from '@/components/binary-install'
import { LangSwitch } from '@/components/lang-switch'
import { SlidingTabIndicator } from '@/components/sliding-tab-indicator'
import { usePlatform, type InstallMethod } from '@/hooks/use-platform'
import { useLanguageSwitch } from '@/hooks/use-language-switch'
import { INSTALL_TABS, getInstallTabDirection, type InstallTab } from '@/lib/tab-motion'
import { i18n } from '@/lib/i18n'

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

// Triggers stay grey; the green active text + underline are drawn by the sliding
// SlidingTabIndicator overlay so they animate across instead of snapping. after:hidden
// suppresses the ui kit's default grey active underline (it would show below ours).
const TAB_TRIGGER_CLS = 'flex-1 h-full rounded-none after:hidden'

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
  return (
    <I18nProvider i18n={i18n}>
      <AppContent />
    </I18nProvider>
  )
}

function AppContent() {
  const { t } = useLingui()
  const platform = usePlatform()
  const { methods, otherMethods, sourceMethod, allBinaries } = platform
  const prefersReducedMotion = useReducedMotion()
  const [tab, setTab] = useState<InstallTab>('command')
  const [slideDirection, setSlideDirection] = useState(0)
  const [mirror, setMirror] = useState(false)
  const [otherMethodsOpen, setOtherMethodsOpen] = useState(false)

  const slideTransition = prefersReducedMotion ? NO_MOTION : SLIDE_SPRING
  const heightTransition = prefersReducedMotion ? NO_MOTION : HEIGHT_EASE

  const { fadeControls, switchLang } = useLanguageSwitch()
  // macOS on Safari/Firefox exposes no CPU architecture, so detection lands in the
  // 'ready' state with no concrete binary. Only then do we offer an explicit Apple
  // Silicon / Intel choice; once the arch is known (Chromium via UA Client Hints) the
  // detected single binary is shown instead. This is the sole ready+null-binary case.
  const showMacOSDownloadChoices = platform.state === 'ready' && platform.binary === null
  const otherPlatformsTitle = t`其他平台`

  function handleTabChange(value: string) {
    const next = value as InstallTab

    setSlideDirection(getInstallTabDirection(tab, next))
    setTab(next)
  }

  const panels: Record<InstallTab, ReactElement> = {
    command: (
      <>
        {platform.state === 'ready' && (
          <>
            <div className="p-6 text-center">
              <p className="text-base text-gray-500 dark:text-gray-400 mb-4">{i18n._(platform.info.hint)}</p>
              <CodeBlock command={mirror ? platform.info.mirrorCommand : platform.info.command} primary />
              <p className="mt-4 text-sm text-gray-400 dark:text-gray-500"><Trans>检测到你的平台：{platform.info.label}</Trans></p>
              {platform.info.warning && (
                <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">⚠ {i18n._(platform.info.warning)}</p>
              )}
            </div>
            {otherMethods.length > 0 && (
              <CollapsibleSection title={otherPlatformsTitle} initial={false}>
                <div className="mt-3"><MethodsList items={otherMethods} mirror={mirror} /></div>
              </CollapsibleSection>
            )}
          </>
        )}

        {platform.state === 'unknown' && (
          <>
            <div className="p-6 text-center">
              <p className="text-base text-gray-500 dark:text-gray-400"><Trans>无法识别你的平台，以下是所有支持的安装方式。</Trans></p>
            </div>
            <div className="px-6 pb-6"><MethodsList items={methods} mirror={mirror} /></div>
          </>
        )}

        {platform.state === 'unsupported' && <div className="px-6 py-5"><MethodsList items={methods} mirror={mirror} /></div>}
      </>
    ),
    download: (
      <BinaryInstall
        binary={platform.binary}
        allBinaries={allBinaries}
        mirror={mirror}
        showMacOSChoices={showMacOSDownloadChoices}
      />
    ),
    source: (
      <div className="p-6 text-center">
        <p className="text-base text-gray-500 dark:text-gray-400 mb-4"><Trans>使用 Go 从源码编译安装：</Trans></p>
        <CodeBlock command={mirror && sourceMethod.mirrorCommand ? sourceMethod.mirrorCommand : sourceMethod.command} primary />
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500"><Trans>需要本机已安装 Go 环境。</Trans></p>
      </div>
    ),
  }

  // Labels keyed by tab id; the ordered list itself comes from INSTALL_TABS (the single
  // source of truth that also types InstallTab and drives slide direction).
  const installTabLabels: Record<InstallTab, ReactElement> = {
    command: <Trans>命令安装</Trans>,
    download: <Trans>下载安装</Trans>,
    source: <Trans>编译安装</Trans>,
  }
  const installTabs = INSTALL_TABS.map(value => ({ value, label: installTabLabels[value] }))

  const installCard = (
    <>
      <Tabs value={tab} onValueChange={handleTabChange} className="flex-col gap-0">
        <TabsList
          variant="line"
          className="relative w-full h-11 p-0 border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/50 rounded-none gap-0"
        >
          {installTabs.map(t => (
            <TabsTrigger
              key={t.value}
              value={t.value}
              id={`install-tab-${t.value}`}
              aria-controls={`install-panel-${t.value}`}
              className={TAB_TRIGGER_CLS}
            >
              {t.label}
            </TabsTrigger>
          ))}
          <SlidingTabIndicator tabs={installTabs} activeValue={tab} direction={slideDirection} transition={slideTransition} />
        </TabsList>

        {/* Only the active panel is mounted; the card's `layout` animation smoothly
            resizes the height when switching tabs (no shared grid, so no cross-tab
            height bleed or stutter). */}
        <div className="relative overflow-hidden">
          <AnimatePresence initial={false} custom={slideDirection} mode="popLayout">
            <motion.div
              key={tab}
              role="tabpanel"
              id={`install-panel-${tab}`}
              aria-labelledby={`install-tab-${tab}`}
              custom={slideDirection}
              variants={panelVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className={`${TAB_CONTENT_CLS} w-full`}
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
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300"><Trans>使用镜像源</Trans></span>
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
              {mirror ? t`GitCode · 镜像源` : t`GitHub · 默认源`}
            </motion.span>
          </AnimatePresence>
        </span>
      </motion.label>
    </>
  )

  return (
    <LayoutGroup id="install-layout">
      <motion.div animate={fadeControls} className="relative max-w-2xl mx-auto px-4 md:px-6 py-12 md:py-24 w-full">
        <LangSwitch onSelect={switchLang} transition={heightTransition} layoutDependency={tab} />
        <motion.header layout="position" layoutDependency={tab} transition={heightTransition}>
          <h1 className="text-6xl sm:text-7xl md:text-8xl" style={{ fontFamily: '"Patua One", serif', fontWeight: 400 }}>
            <span className="cj-gradient" style={{ paddingBottom: '0.15em', display: 'inline-block' }}>cjv</span>
          </h1>
          <p className="mt-6 text-lg md:text-2xl text-gray-500 dark:text-gray-400">
            <Trans>
              <a href="https://cangjie-lang.cn/" target="_blank" rel="noopener noreferrer" className="text-cj dark:text-cj-light hover:underline mr-1">仓颉</a>编程语言 SDK 工具链管理器
            </Trans>
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
                  <Trans>cjv 暂不支持 <strong className="text-gray-700 dark:text-gray-300">{platform.info.label}</strong> 平台。</Trans>
                </p>
                {platform.info.reason === 'arch' ? (
                  <>
                    <p className="mt-2 text-sm text-gray-400 dark:text-gray-500"><Trans>该架构暂无预编译版本。</Trans></p>
                    <p className="mt-3 text-sm text-gray-400 dark:text-gray-500">
                      <Trans>
                        你可以尝试 x86_64（amd64）版本（多数情况下可通过系统模拟运行），或从{' '}
                        <a href="https://github.com/Zxilly/cjv/releases" target="_blank" rel="noopener noreferrer" className="text-cj dark:text-cj-light hover:underline">Releases</a>{' '}
                        手动下载。
                      </Trans>
                    </p>
                  </>
                ) : (
                  <>
                    <p className="mt-2 text-sm text-gray-400 dark:text-gray-500"><Trans>cjv 目前仅在 Linux、macOS 和 Windows 桌面系统上提供。</Trans></p>
                    <p className="mt-3 text-sm text-gray-400 dark:text-gray-500">
                      <Trans>
                        请在桌面设备上访问此页面，或查看{' '}
                        <a href="https://github.com/Zxilly/cjv" target="_blank" rel="noopener noreferrer" className="text-cj dark:text-cj-light hover:underline">GitHub</a>{' '}
                        了解更多。
                      </Trans>
                    </p>
                  </>
                )}
              </div>
              <Collapsible
                open={otherMethodsOpen}
                onOpenChange={setOtherMethodsOpen}
                className="border-t border-gray-200 dark:border-gray-800"
              >
                <CollapsibleTrigger className="group cursor-pointer px-6 py-4 w-full text-left text-sm text-gray-500 dark:text-gray-400 hover:text-cj dark:hover:text-cj-light flex items-center gap-1.5">
                  <ChevronRight className="size-4 transition-transform duration-200 group-data-[state=open]:rotate-90" />
                  <span className="group-hover:underline"><Trans>查看其他平台的安装方式</Trans></span>
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
          <a href={`/book/user-guide/${i18n.locale === 'en' ? 'en' : 'zh-CN'}/`} className="hover:text-cj dark:hover:text-cj-light"><Trans>文档</Trans></a>
          <span>·</span>
          <a href="https://github.com/Zxilly/cjv" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light">GitHub</a>
          <span>·</span>
          <a href="https://github.com/Zxilly/cjv/releases" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light">Releases</a>
          <span>·</span>
          <a href="https://cangjie-lang.cn/" target="_blank" rel="noopener noreferrer" className="hover:text-cj dark:hover:text-cj-light"><Trans>仓颉官网</Trans></a>
        </motion.nav>
      </motion.div>
    </LayoutGroup>
  )
}
