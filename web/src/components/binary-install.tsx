import { Download } from 'lucide-react'
import { CollapsibleSection } from './collapsible-section'
import type { BinaryInfo } from '@/hooks/use-platform'

interface BinaryInstallProps {
  binary: BinaryInfo | null
  allBinaries: BinaryInfo[]
  mirror: boolean
  onUseCommandInstall?: () => void
}

const platformKey = (b: BinaryInfo) => `${b.goos}_${b.goarch}`

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
              href={mirror ? b.mirrorUrl : b.officialUrl}
              download={b.binaryName}
              rel="noopener"
              className="group/dl flex items-center justify-between gap-3 px-2 py-2 rounded hover:bg-gray-100/70 dark:hover:bg-gray-800/40 focus-visible:outline-none focus-visible:bg-gray-100/70 dark:focus-visible:bg-gray-800/40 transition-colors"
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
          或从{' '}
          <a
            href={mirror ? 'https://gitcode.com/Zxilly/cjv/releases' : 'https://github.com/Zxilly/cjv/releases'}
            target="_blank"
            rel="noopener noreferrer"
            className="text-cj dark:text-cj-light hover:underline"
          >
            {mirror ? 'GitCode Releases' : 'GitHub Releases'}
          </a>{' '}
          下载完整发布包。
        </p>
      )}
    </>
  )
}

export function BinaryInstall({ binary, allBinaries, mirror, onUseCommandInstall }: BinaryInstallProps) {
  if (!binary) {
    return (
      <>
        <div className="px-6 pt-6 pb-2 text-center">
          <p className="text-base text-gray-500 dark:text-gray-400">请手动选择对应平台的二进制：</p>
        </div>
        <div className="px-6 pb-4">
          <ManualDownloadList binaries={allBinaries} mirror={mirror} />
        </div>
        {onUseCommandInstall && (
          <div className="px-6 py-3 text-center">
            <button
              type="button"
              onClick={onUseCommandInstall}
              className="text-xs text-gray-400 dark:text-gray-500 hover:text-cj dark:hover:text-cj-light hover:underline cursor-pointer focus-visible:outline-none focus-visible:underline"
            >
              不确定？切换到命令安装 →
            </button>
          </div>
        )}
      </>
    )
  }

  const primaryUrl = mirror ? binary.mirrorUrl : binary.officialUrl
  const others = allBinaries.filter(b => b !== binary)

  return (
    <>
      <div className="p-6 text-center">
        <p className="text-base text-gray-500 dark:text-gray-400 mb-3">
          要安装 cjv，下载并运行
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
        <p className="text-base text-gray-500 dark:text-gray-400 mt-3">然后按屏幕提示完成安装。</p>
        {onUseCommandInstall && (
          <p className="mt-2 text-xs text-gray-400 dark:text-gray-500">
            或使用{' '}
            <button
              type="button"
              onClick={onUseCommandInstall}
              className="text-cj dark:text-cj-light hover:underline cursor-pointer focus-visible:outline-none focus-visible:underline"
            >
              命令安装
            </button>
            {' '}（自动校验并配置 PATH）
          </p>
        )}
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">检测到你的平台：{binary.label}</p>
        {binary.warning && (
          <p className="mt-3 text-sm text-amber-600 dark:text-amber-400">⚠ {binary.warning}</p>
        )}
      </div>

      <CollapsibleSection title="其他平台" initial={false}>
        <div className="mt-2">
          <ManualDownloadList binaries={others} mirror={mirror} />
        </div>
      </CollapsibleSection>
    </>
  )
}
