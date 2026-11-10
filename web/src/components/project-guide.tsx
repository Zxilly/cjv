import { useLingui } from '@lingui/react/macro'

const content = {
  zh: {
    title: '从安装到项目协作',
    links: [
      ['安装仓颉 SDK', '选择适合当前系统的安装方式，验证 cjc 和 cjpm。', 'installation/'],
      ['固定项目版本', '用 cangjie-sdk.toml 声明版本、stdx 组件和目标 SDK。', 'toolchain-file.html'],
      ['配置 CI 与离线环境', '在自动化构建中校验版本、复用缓存并控制联网行为。', 'automation.html'],
    ],
    note: '各平台可用的 SDK 以分发源为准。cjv 管理工具链，项目构建与依赖管理仍由 cjpm 完成。',
    story: '为什么用 cjv 管理多个 SDK？',
    support: '觉得有用？在 GitHub 上 Star 支持项目',
  },
  en: {
    title: 'From installation to team projects',
    links: [
      ['Install the Cangjie SDK', 'Choose an installation method for your system and verify cjc and cjpm.', 'installation/'],
      ['Pin a project toolchain', 'Declare versions, stdx components and target SDKs in cangjie-sdk.toml.', 'toolchain-file.html'],
      ['Set up CI and offline builds', 'Check versions, reuse caches and control network access in automated builds.', 'automation.html'],
    ],
    note: 'SDK availability depends on the distribution source. cjv manages toolchains; cjpm still handles project builds and dependencies.',
    story: 'Why manage multiple SDKs with cjv? (Chinese)',
    support: 'Find it useful? Star cjv on GitHub',
  },
}

export function ProjectGuide() {
  const { i18n } = useLingui()
  const lang = i18n.locale === 'en' ? 'en' : 'zh'
  const copy = content[lang]
  const base = `/book/user-guide/${lang === 'en' ? 'en' : 'zh-CN'}/`
  return (
    <section aria-labelledby="project-guide" className="mt-12 border-t border-gray-200 dark:border-gray-800 pt-8">
      <h2 id="project-guide" className="text-lg font-medium text-gray-800 dark:text-gray-200">{copy.title}</h2>
      <ul className="mt-5 space-y-5">
        {copy.links.map(([title, description, path]) => (
          <li key={path}>
            <a href={base + path} data-umami-event="open-guide" data-umami-event-guide={path} className="text-cj dark:text-cj-light hover:underline font-medium">{title}</a>
            <p className="mt-1 text-sm leading-relaxed text-gray-500 dark:text-gray-400">{description}</p>
          </li>
        ))}
      </ul>
      <p className="mt-6 text-sm leading-relaxed text-gray-500 dark:text-gray-400">{copy.note}</p>
      <p className="mt-5 text-sm"><a href="https://learningman.top/archives/462" className="text-cj dark:text-cj-light hover:underline">{copy.story}</a></p>
      <p className="mt-3 text-sm"><a href="https://github.com/Zxilly/cjv" data-umami-event="github-repository" className="text-cj dark:text-cj-light hover:underline">{copy.support}</a></p>
    </section>
  )
}
