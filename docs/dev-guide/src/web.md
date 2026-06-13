# 落地页

落地页是 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的根页面,源码在仓库的 `web/` 目录。它是一个静态单页应用,核心职责是检测访问者的操作系统和 CPU 架构,然后给出最合适的安装命令或下载链接。页面没有后端,所有逻辑都在浏览器里跑,构建产物是一堆静态文件,由 GitHub Pages 托管。

文档站和落地页一起部署在同一个 Pages 站点上:落地页占据站点根路径,用户手册和本开发指南被构建进 `web/dist/book/` 下作为子路径。这一节只讲落地页本身,文档站的构建见 [documentation.md](documentation.md),整套 Pages 部署流程见 [ci.md](ci.md)。

## 技术栈

落地页用 Vite 打包,React 19 写界面,Tailwind CSS v4 做样式,framer-motion 做动效。包管理器是 pnpm,版本锁在 `web/package.json` 的 `packageManager` 字段里(`pnpm@10`),CI 用的 Node 是 22。

`web/vite.config.ts` 把三个插件串在一起:

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { lingui } from '@lingui/vite-plugin'
import path from 'node:path'

export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  plugins: [
    react({ babel: { plugins: ['@lingui/babel-plugin-lingui-macro'] } }),
    tailwindcss(),
    lingui(),
  ],
})
```

`@vitejs/plugin-react` 里挂了 `@lingui/babel-plugin-lingui-macro`,这样 Lingui 的宏(下面会讲)能在编译期展开。`@` 是 `src/` 的路径别名,导入时写 `@/hooks/use-platform` 这种形式;同样的别名在 `web/vitest.config.ts` 里也配了一份,跑测试时才能解析。

Tailwind v4 不需要 `tailwind.config.js`,主题直接写在 CSS 里。入口样式是 `web/src/style.css`,顶部 `@import "tailwindcss"` 引入框架,`@theme` 块定义自定义 token,比如品牌色 `--color-cj`(对应工具类 `text-cj`、`bg-cj`)和字体栈。深色模式用 `@custom-variant dark` 配合 `prefers-color-scheme` 媒体查询,没有手动切换开关。字体通过 `@fontsource` 包以 `@import` 的方式打进产物,不依赖外部 CDN。

界面结构集中在 `web/src/App.tsx`,这是整个页面的主组件:标题、安装卡片(命令安装 / 下载安装 / 编译安装三个标签页)、镜像源开关、底部导航都在这里。可复用的小组件在 `web/src/components/`,其中 `components/ui/` 下是基于 `radix-ui` 封装的无样式原语(`tabs`、`switch`、`collapsible` 等)。framer-motion 负责标签页切换的滑动、卡片高度变化的过渡这些动效,并且统一通过 `useReducedMotion` 尊重系统的"减少动态效果"偏好,开启后所有过渡降级为零时长。

## 国际化

落地页用 [Lingui](https://lingui.dev) 做 i18n。源语言是简体中文,英文是翻译,这跟整个项目保持一致:你在代码里直接写中文,英文译文由 po catalog 提供。配置在 `web/lingui.config.ts`:

```ts
import { defineConfig } from '@lingui/cli'

export default defineConfig({
  sourceLocale: 'zh',
  locales: ['zh', 'en'],
  catalogs: [
    {
      path: '<rootDir>/src/locales/{locale}/messages',
      include: ['<rootDir>/src'],
    },
  ],
})
```

catalog 落在 `web/src/locales/zh/messages.po` 和 `web/src/locales/en/messages.po`。因为源语言就是中文,`zh` catalog 里每条的 `msgstr` 和 `msgid` 是一样的;`en` catalog 里 `msgstr` 才是真正的英文翻译。

### 在代码里写可翻译字符串

Lingui 通过宏在编译期收集字符串,所以你写的是宏,而不是手动维护 ID。落地页里用到三种,按场景选:

JSX 里用 `<Trans>`,从 `@lingui/react/macro` 导入。它能包裹带子元素的文案,占位的标签会被编号:

```tsx
import { Trans } from '@lingui/react/macro'

<Trans>使用 Go 从源码编译安装：</Trans>
```

需要运行时拿到字符串(比如传给 `title` 属性或做条件判断)时,用 `useLingui()` 返回的 `t` 模板标签,同样来自 `@lingui/react/macro`:

```tsx
import { useLingui } from '@lingui/react/macro'

const { t } = useLingui()
const title = t`其他平台`
```

如果字符串要在组件外、模块顶层定义(`web/src/hooks/use-platform.ts` 里就有不少这种),用 `@lingui/core/macro` 的 `msg`,它产出一个 `MessageDescriptor`,延迟到渲染时再用 `i18n._(descriptor)` 求值:

```ts
import { msg } from '@lingui/core/macro'

const MAC_X86_WARNING = msg`部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。`
```

### 加了字符串之后

写完新的中文文案后,跑提取命令把它们同步进 catalog:

```bash
cd web
pnpm i18n:extract
```

这会扫描 `src/` 下所有宏,把新 `msgid` 写进两个 `messages.po`。新条目在 `zh` catalog 里 `msgstr` 自动等于中文原文;在 `en` catalog 里 `msgstr` 是空的,需要你(或后续的翻译流程)填上英文。提交前请把 `en` 的译文补齐,不要留空的 `msgstr`。`pnpm i18n:compile` 是另一个脚本,生产构建时由 Vite 的 Lingui 插件直接处理 `.po`,日常开发一般用不到它。

运行时的语言加载在 `web/src/lib/i18n.ts`:模块加载时调用 `detectLang()` 选语言(优先 `localStorage` 里存的 `cjv-lang`,否则看 `navigator.language` 是不是 `zh`,默认回退英文),然后 `activateLang` 激活对应 catalog。`web/src/App.tsx` 顶层用 `<I18nProvider>` 把 `i18n` 实例注入 React 树,右上角的语言切换按钮调 `activateLang` + `persistLang` 实时切换并记住选择。

## 平台检测

平台检测是落地页的核心,逻辑全在 `web/src/hooks/use-platform.ts`,纯函数加一个 `usePlatform` hook。它要回答的问题是:访问者在什么系统、什么架构上,该给他哪条安装命令。

检测分两步。首次渲染时同步算一个初步结果(`computeBrowserPlatformResult`),用 `navigator.platform`、`navigator.userAgent` 这些立刻能拿到的信息,配合 `ua-parser-modern` 解析。挂载后再异步调一次(`detectBrowserPlatformResult`):如果浏览器支持 UA Client Hints(`navigator.userAgentData.getHighEntropyValues`),就请求更精确的 `architecture`、`bitness`、`platform`,拿到后重算。两步之间用 `samePlatformResult` 比较用户可见字段,相同就跳过重渲染。

检测结果是一个判别联合 `PlatformResult`,`state` 有三种:

- `ready`:识别出受支持的平台。多数情况下还附带一个具体的 `binary`(可直接下载的链接)。有一个例外:macOS 上的 Safari 和 Firefox 不暴露 CPU 架构,这时仍然是 `ready`(因为 `install.sh` 会自己解析架构),但 `binary` 为 `null`,界面改为让用户在 Apple Silicon 和 Intel 之间二选一。
- `unsupported`:已知是手机系统(iOS / Android / HarmonyOS),或者是已知桌面系统但架构没有预编译版本(比如 Windows arm64)。`info.reason` 会是 `'mobile'` 或 `'arch'`,界面据此给不同的建议。
- `unknown`:认不出来,直接把所有平台的安装方式都列出来。

支持哪些平台不是在前端写死的。`web/src/generated/platforms.ts` 是生成文件,由 `scripts/gen-platform-surfaces.go` 从 Go 端的 `internal/target` 包导出,顶部带 `DO NOT EDIT` 标记。这样前端展示的平台列表和 cjv 真正支持的构建目标永远一致,改了 Go 端的支持矩阵后跑 `go generate ./...`(hook 在 `internal/target/catalog.go`)就能同步过来,不用手动维护两份。`use-platform.ts` 里的 `PLATFORM_PRESENTATION` 只负责给这些平台补上展示用的 label、提示文案和安装命令。

HarmonyOS 的识别有一段专门处理:`ua-parser-modern` 只在 UA 同时带 `android` 和 `harmonyos` 标记时才报 `HarmonyOS`,而 HarmonyOS NEXT 去掉了 `android` 标记会被误判,所以代码额外在原始 UA 里匹配 `harmonyos`/`openharmony` 兜底。iPadOS 的"桌面模式"会把自己伪装成 macOS,代码用 `platform === 'MacIntel'` 加 `maxTouchPoints > 1` 把它识别回 iOS。这些都是有注释的实际边界情况,改这块时留意别破坏。

为什么不在服务端直接读架构、而要在浏览器里做这一整套猜测,背景见自动记忆里关于 macOS 架构检测的笔记,核心结论是浏览器无法可靠读到 Mac 的 CPU 架构,所以才有 `install.sh` 加二选一下载的兜底设计。

## 本地开发

所有命令都在 `web/` 目录下跑。首次进来先装依赖:

```bash
cd web
pnpm install
```

常用脚本(定义在 `web/package.json`):

```bash
pnpm dev      # 启动 Vite 开发服务器,带热更新
pnpm build    # tsc -b 类型检查 + vite build,产物在 web/dist
pnpm preview  # 本地预览 web/dist 里已构建的产物
```

`pnpm build` 会先跑 `tsc -b` 做一遍完整类型检查再打包,所以类型错误会让构建直接失败。

测试用 Vitest,跑在真实浏览器里(`@vitest/browser` + Playwright),不是 jsdom。配置在 `web/vitest.config.ts`,默认浏览器是 chromium,可以用环境变量 `VITEST_BROWSER` 切到 `firefox` 或 `webkit`。常用脚本:

```bash
pnpm test       # 单次跑全部测试
pnpm test:watch # watch 模式
pnpm coverage   # 跑测试并出覆盖率报告(阈值 80%)
```

测试是在浏览器里跑的,首次跑之前需要装好 Playwright 的浏览器:`pnpm exec playwright install --with-deps chromium`。`pnpm test:integration` 单独跑 `src/App.platform.integration.test.tsx`,这是用真实浏览器 UA 验证端到端平台检测的集成测试,CI 里会在 chromium、firefox、webkit 三个引擎、跨多个 runner 操作系统的矩阵下跑它。测试的整体约定见 [testing.md](testing.md)。

CI 侧,`.github/workflows/ci.yml` 有 `web-build`、`web-test`、`web-coverage`、`web-browser-integration` 几个 job,分别对应上面这些命令。部署由 `.github/workflows/pages.yml` 负责:它先 `pnpm build` 出 `web/dist`,再把各平台的 `cjv-init` 二进制解包到 `web/dist/dl/` 供页面的下载链接使用,接着把用户手册和开发指南构建进 `web/dist/book/`,最后整个 `web/dist` 作为 Pages artifact 发布。详见 [ci.md](ci.md)。
