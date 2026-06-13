import { Apple, Cpu, Download, type LucideIcon } from 'lucide-react'
import { Trans, useLingui } from '@lingui/react/macro'
import { CollapsibleSection } from './collapsible-section'
import { i18n } from '@/lib/i18n'
import type { BinaryInfo } from '@/hooks/use-platform'

interface BinaryInstallProps {
  binary: BinaryInfo | null
  allBinaries: BinaryInfo[]
  mirror: boolean
  showMacOSChoices?: boolean
}

const platformKey = (b: BinaryInfo) => `${b.goos}_${b.goarch}`
const binaryUrl = (b: BinaryInfo, mirror: boolean) => mirror ? b.mirrorUrl : b.officialUrl

const DL_ROW_CLS =
  'group/dl flex items-center justify-between gap-3 px-2 py-2 rounded hover:bg-gray-100/70 dark:hover:bg-gray-800/40 focus-visible:outline-none focus-visible:bg-gray-100/70 dark:focus-visible:bg-gray-800/40 transition-colors'

// Per-arch presentation for the macOS chip chooser. Apple Silicon carries the brand
// emerald (it is the common case on modern Macs); Intel stays neutral and pairs with
// the amber SDK caveat below.
const MAC_CHOICE_META: Record<BinaryInfo['goarch'], {
  title: string
  Icon: LucideIcon
  card: string
  tile: string
  pill: string
}> = {
  arm64: {
    title: 'Apple Silicon',
    Icon: Apple,
    card: 'hover:border-cj/40 dark:hover:border-cj-light/45',
    tile: 'bg-gradient-to-br from-cj/15 to-cj/5 text-cj ring-1 ring-cj/20 group-hover/card:ring-cj/45 dark:from-cj-light/15 dark:to-cj-light/5 dark:text-cj-light dark:ring-cj-light/25 dark:group-hover/card:ring-cj-light/50',
    pill: 'group-hover/card:bg-cj group-hover/card:text-white dark:group-hover/card:bg-cj-light dark:group-hover/card:text-gray-900',
  },
  amd64: {
    title: 'Intel',
    Icon: Cpu,
    card: 'hover:border-gray-300 dark:hover:border-gray-600',
    tile: 'bg-gradient-to-br from-gray-100 to-gray-50 text-gray-500 ring-1 ring-gray-200 group-hover/card:ring-gray-300 dark:from-gray-800 dark:to-gray-800/40 dark:text-gray-300 dark:ring-gray-700 dark:group-hover/card:ring-gray-600',
    pill: 'group-hover/card:bg-gray-800 group-hover/card:text-white dark:group-hover/card:bg-gray-200 dark:group-hover/card:text-gray-900',
  },
}

function ManualDownloadList({
  binaries,
  mirror,
  showFullReleaseHint = true,
}: {
  binaries: BinaryInfo[]
  mirror: boolean
  showFullReleaseHint?: boolean
}) {
  return (
    <>
      <ul className="-mx-2">
        {binaries.map(b => (
          <li key={platformKey(b)}>
            <a
              href={binaryUrl(b, mirror)}
              download={b.binaryName}
              rel="noopener"
              className={DL_ROW_CLS}
            >
              <span className="text-sm text-gray-700 dark:text-gray-300">{b.label}</span>
              <span className="inline-flex items-center gap-1.5 font-mono text-sm text-cj dark:text-cj-light group-hover/dl:underline group-focus-visible/dl:underline underline-offset-4">
                <Download className="size-3.5 opacity-70 group-hover/dl:opacity-100 transition-opacity" strokeWidth={2} />
                {b.binaryName}
              </span>
            </a>
          </li>
        ))}
      </ul>
      {showFullReleaseHint && (
        <p className="mt-3 px-2 text-sm text-gray-400">
          <Trans>
            或{' '}
            <a
              href={mirror ? 'https://gitcode.com/Zxilly/cjv/releases' : 'https://github.com/Zxilly/cjv/releases'}
              target="_blank"
              rel="noopener noreferrer"
              className="text-cj dark:text-cj-light hover:underline"
            >
              {mirror ? 'GitCode Releases' : 'GitHub Releases'}
            </a>{' '}
            下载完整发布包。
          </Trans>
        </p>
      )}
    </>
  )
}

function MacOSDownloadChoices({
  allBinaries,
  mirror,
}: {
  allBinaries: BinaryInfo[]
  mirror: boolean
}) {
  const { t } = useLingui()
  // ALL_BINARIES always carries both darwin variants, ordered arm64 then amd64.
  const choices = allBinaries.filter(b => b.goos === 'darwin')
  const otherBinaries = allBinaries.filter(b => b.goos !== 'darwin')
  const warning = choices.find(b => b.warning)?.warning

  return (
    <>
      <div className="px-6 pt-6 pb-5">
        <div className="mb-4 text-center">
          <p className="text-base text-gray-600 dark:text-gray-300"><Trans>选择你的 Mac 芯片，下载 cjv 安装器</Trans></p>
          <p className="mt-1 text-xs text-gray-400 dark:text-gray-500"><Trans>不确定？打开 Apple 菜单 →「关于本机」查看芯片型号</Trans></p>
        </div>
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {choices.map(b => {
            const meta = MAC_CHOICE_META[b.goarch]
            const { title, Icon } = meta
            const detail = b.goarch === 'arm64' ? t`M1 / M2 / M3 及更新机型` : t`2020 年前的 Intel 机型`
            const badge = b.goarch === 'arm64' ? t`常见` : undefined
            return (
              <li key={platformKey(b)}>
                <a
                  href={binaryUrl(b, mirror)}
                  download={b.binaryName}
                  rel="noopener"
                  className={`group/card relative flex h-full flex-col gap-4 rounded-xl border border-gray-200 bg-white p-4 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-gray-900/[0.06] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cj/40 dark:border-gray-800 dark:bg-gray-900/40 dark:hover:shadow-black/40 ${meta.card}`}
                >
                  {badge && (
                    <span className="absolute right-3 top-3 rounded-full bg-cj/10 px-2 py-0.5 text-[10px] font-medium tracking-wide text-cj dark:bg-cj-light/15 dark:text-cj-light">
                      {badge}
                    </span>
                  )}
                  <span className="flex items-center gap-3">
                    <span className={`inline-flex size-11 shrink-0 items-center justify-center rounded-xl transition-all duration-200 ${meta.tile}`}>
                      <Icon className="size-5 transition-transform duration-200 group-hover/card:scale-110" strokeWidth={1.75} />
                    </span>
                    <span className="min-w-0">
                      <span className="block text-base font-semibold text-gray-900 dark:text-gray-100">{title}</span>
                      <span className="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{detail}</span>
                    </span>
                  </span>
                  <span className={`mt-auto flex items-center justify-between gap-2 rounded-lg bg-gray-50 px-3 py-2 font-mono text-sm text-gray-600 transition-colors duration-200 dark:bg-gray-800/50 dark:text-gray-300 ${meta.pill}`}>
                    <span className="truncate">{b.binaryName}</span>
                    <Download className="size-4 shrink-0 transition-transform duration-200 group-hover/card:translate-y-0.5" strokeWidth={2} />
                  </span>
                </a>
              </li>
            )
          })}
        </ul>
        {warning && (
          <p className="mt-4 rounded-lg border border-amber-300/50 bg-amber-50/70 px-3.5 py-2.5 text-xs leading-relaxed text-amber-700 dark:border-amber-500/25 dark:bg-amber-500/10 dark:text-amber-400/90">
            ⚠ {i18n._(warning)}
          </p>
        )}
      </div>

      <CollapsibleSection title={t`其他平台`} initial={false}>
        <div className="mt-2">
          <ManualDownloadList binaries={otherBinaries} mirror={mirror} />
        </div>
      </CollapsibleSection>
    </>
  )
}

export function BinaryInstall({ binary, allBinaries, mirror, showMacOSChoices = false }: BinaryInstallProps) {
  const { t } = useLingui()

  if (showMacOSChoices) {
    return <MacOSDownloadChoices allBinaries={allBinaries} mirror={mirror} />
  }

  if (!binary) {
    return (
      <>
        <div className="px-6 pt-6 pb-2 text-center">
          <p className="text-base text-gray-500 dark:text-gray-400"><Trans>请手动选择对应平台的二进制：</Trans></p>
        </div>
        <div className="px-6 pb-4">
          <ManualDownloadList binaries={allBinaries} mirror={mirror} />
        </div>
      </>
    )
  }

  const primaryUrl = binaryUrl(binary, mirror)
  const others = allBinaries.filter(b => b !== binary)

  return (
    <>
      <div className="p-6 text-center">
        <p className="text-base text-gray-500 dark:text-gray-400 mb-3">
          <Trans>要安装 cjv，下载并运行</Trans>
        </p>
        <a
          href={primaryUrl}
          download={binary.binaryName}
          rel="noopener"
          className="inline-flex items-center gap-1.5 font-mono text-lg md:text-xl font-medium text-cj dark:text-cj-light underline decoration-2 underline-offset-[5px] hover:decoration-[3px] hover:underline-offset-[7px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cj/40 dark:focus-visible:ring-cj-light/40 rounded-sm transition-all"
        >
          <Download className="size-4 md:size-5" strokeWidth={2.25} />
          {binary.binaryName}
        </a>
        <p className="text-base text-gray-500 dark:text-gray-400 mt-3"><Trans>然后按屏幕提示完成安装。</Trans></p>
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500"><Trans>检测到你的平台：{binary.label}</Trans></p>
        {binary.warning && (
          <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">⚠ {i18n._(binary.warning)}</p>
        )}
      </div>

      <CollapsibleSection title={t`其他平台`} initial={false}>
        <div className="mt-2">
          <ManualDownloadList binaries={others} mirror={mirror} />
        </div>
      </CollapsibleSection>
    </>
  )
}
