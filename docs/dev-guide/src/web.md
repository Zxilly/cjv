# 落地页

`web/` 是部署在 [cjv.zxilly.com](https://cjv.zxilly.com) 和 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的静态落地页，提供安装命令和各平台二进制下载。前端使用 React Router Framework Mode、Vite、TypeScript、Tailwind CSS 和 Lingui；`web/package.json` 声明依赖范围及 pnpm 版本，`web/pnpm-lock.yaml` 锁定依赖版本。

## 开发与文件位置

在 `web/` 目录下启动本地开发服务器：

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` 会生成路由类型、执行 TypeScript 检查，再由 React Router 官方构建流程生成中文 `/` 与英文 `/en/` 的完整 HTML。`react-router.config.ts` 使用 `ssr: false` 和 根据 Blog 清单生成的 `prerender` 路径，静态产物位于 `web/dist/client/`。`pnpm preview` 用于预览这些生产产物。需要 Node.js 22.22.0 或更新版本，线上无需运行 React SSR 服务。

预渲染与首次 hydration 都使用通用安装方式，接管后再识别浏览器平台、启用边缘节点给出的镜像默认值。不要在首次渲染时读取浏览器平台或用户偏好，否则会产生 hydration 不匹配。页面正文、命令和文档链接在禁用 JavaScript 时也可读取。

`src/lib/seo.ts` 定义两种语言的页面元信息，首页底部导航提供对应语言的文档与 Blog 入口。两域名的 canonical 与 hreflang 统一指向 `.com` 对应页面；`.dev` 继续提供安装与下载。文档由 `mdbook-sitemap` 后端生成章节 sitemap，`scripts/site_seo.py` 补充逐页元信息、规范化地址，并将插件产物、中英文首页和 Blog 页面合并为全站 sitemap。

| 文件或目录 | 内容 |
| --- | --- |
| `react-router.config.ts`、`src/routes.ts` | 官方预渲染配置与路由表 |
| `src/root.tsx` | HTML 文档、SEO 元信息与 hydration 入口组件 |
| `src/App.tsx` | 安装页面布局、镜像源切换与下载交互 |
| `src/hooks/use-platform.ts` | 浏览器平台识别与下载建议 |
| `src/generated/platforms.ts` | 由 Go 平台清单（`internal/target/catalog.go`）生成的前端数据 |
| `src/lib/i18n.ts`、`src/locales/` | 语言选择与 Lingui 翻译目录 |
| `src/style.css`、`src/components/` | 主题样式和界面组件 |
| `public/install.sh`、`public/install.ps1` | 公开分发的 Unix 与 PowerShell 安装脚本 |

修改支持的发布平台时，请更新 Go 的 `internal/target/catalog.go` 清单并在仓库根目录运行 `go generate ./...`，不要直接手动编辑生成的 `src/generated/platforms.ts`。

## 平台检测

`use-platform.ts` 先根据同步浏览器信息（`navigator.userAgent` 与 `navigator.platform`）计算初始结果，再尝试通过 UA Client Hints（`navigator.userAgentData`）补充 CPU 架构信息。检测结果分为 `ready`、`unknown` 和 `unsupported` 三种状态：

- **macOS 架构歧义处理** — 当 macOS 浏览器无法可靠报告真实 CPU 架构时，`install.sh` 安装脚本命令仍然可用（脚本会在终端本机检测真实架构），而手动下载面板会同时显示 Apple Silicon 与 Intel 两个选项。
- **移动设备与平板** — 移动操作系统及没有预编译二进制的架构显示为不支持（`unsupported`），同时还处理了 iPadOS 桌面模式的 User-Agent 差异。
- **鸿蒙（HarmonyOS / OpenHarmony）检测** — 鸿蒙检测优先读取 `OpenHarmony` / `HarmonyOS` 标记中的版本号并保留其版本命名空间，绝不将 `ArkWeb`、`HuaweiBrowser` 或 `Chrome` 版本当作系统版本。`Phone` / `Mobile` 表示手机，`Tablet` 表示普通平板（`unsupported`）；`Tablet` 同时包含 `Windows NT` 时识别为二合一设备平板模式，`PC` 或桌面 `Windows NT` 标记识别为鸿蒙电脑。电脑和二合一设备显示 shell 安装命令及原生 `ARM64` / `x86_64` 下载选项；手机、普通平板和无法确认类型的设备保留对应提示。

这些兼容标记及 UA Client Hints 不会将鸿蒙覆盖识别为 Windows 或 Android，也不用于猜测鸿蒙设备的原生 CPU 架构；下载面板会说明如何用 `uname -m` 确认架构，以及如何在终端运行 `cjv-init`。

修改平台识别逻辑后，运行真实浏览器平台测试（`pnpm test:integration` 和 `pnpm test:built`），确认无法识别架构时不会错误推荐单一二进制。测试命令见[测试与检查](testing.md#安装脚本与前端)。

## 翻译与样式

中文（`zh`）是 Lingui 的源语言：

- JSX 元素使用 `@lingui/react/macro` 的 `<Trans>`。
- 组件内字符串表达式使用 `useLingui().t`。
- 组件外消息描述符使用 `@lingui/core/macro` 的 `msg`。

添加或修改前端文案后，提取词条：

```bash
pnpm i18n:extract
```

随后补齐 `src/locales/en/messages.po` 中的新英文译文。生产页面的语言由 URL 确定，语言链接同时更新 URL、正文和元信息，并支持浏览器前进与后退。开发与生产均以路由为准，保存的语言偏好不会覆盖显式 URL。每个页面渲染树使用独立的 Lingui 实例，避免预渲染时串语言。生产构建由 Lingui Vite 插件自动编译词条目录。

视觉主题在 `src/style.css` 中定义，深色模式跟随系统设置，字体由本地依赖打包。界面动效需继续遵循 `useReducedMotion` 的无障碍偏好。

首页保留原始标志、简短介绍、安装面板和底部导航样式，仅在原有文档、GitHub、Releases、仓颉官网链接中增加 Blog 入口。教程目录、项目协作说明和文章推荐放在文档或 Blog 中，避免重复拉长安装页。

## Blog 内容发布

Blog 使用 `/blog/` 和 `/en/blog/`，每篇文章有对应的中英文地址。`src/content/blog/posts.json` 管理 slug、发布日期及双语标题、摘要和分类；正文分别放在 `src/content/blog/zh/` 和 `en/` 下的同名 Markdown 文件中。Blog 的界面词条集中在 `src/lib/blog.ts`，独立于首页的 Lingui 目录。

新增文章时同时提供两个语言版本，正文从 H2 开始，不重复页面的 H1 标题。使用稳定的 slug，文档链接指向 `.com` 的对应语言页面。`react-router.config.ts` 根据文章清单自动预渲染列表和文章；服务端构建模块解析 Markdown，生产环境不需要 Markdown API。每篇文章都有独立的 canonical、hreflang、摘要与 BlogPosting 数据。文档构建将实际生成的 Blog 页面加入全站 sitemap。

Blog 延续首页的绿色品牌、字体与系统深色模式，以单栏正文、目录和可横向滚动的代码块组织阅读。更新后运行 `pnpm test`、`pnpm build` 和 `pnpm test:built`，并检查桌面及手机布局。文章中的 SDK 和 CI 命令仍需在所述环境中验证；网页构建测试不代表 SDK 工作流已经执行。

## 部署内容

在 CI 部署中，`.github/workflows/pages.yml` 会先构建 `web/` 前端，再向 `web/dist/client/` 中放入各平台的 `/dl/` `cjv-init` 安装器和两本双语手册。本地单独执行 Vite 构建不会生成这些额外目录。下载安装器来源见[发布流程](releasing.md#网页安装器)，文档构建输出见[文档站](documentation.md)。
