# 落地页

`web/` 是部署在 [cjv.zxilly.com](https://cjv.zxilly.com) 和 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的静态落地页，提供安装命令和各平台二进制下载。前端使用 React、Vite、TypeScript、Tailwind CSS 和 Lingui；`web/package.json` 声明依赖范围及 pnpm 版本，`web/pnpm-lock.yaml` 锁定依赖版本。

## 开发与文件位置

在 `web/` 目录下启动本地开发服务器：

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` 会运行 TypeScript 类型检查、Vite 客户端构建，再用 `scripts/prerender.mjs` 和 `src/entry-server.tsx` 在构建时渲染 React，生成中文 `/` 与英文 `/en/` 的完整 HTML。`pnpm preview` 用于预览这些生产产物。线上无需运行 React SSR 服务。

预渲染与首次 hydration 都使用通用安装方式，接管后再识别浏览器平台、启用边缘节点给出的镜像默认值。不要在首次渲染时读取浏览器平台或用户偏好，否则会产生 hydration 不匹配。页面正文、命令和文档链接在禁用 JavaScript 时也可读取。

`src/lib/seo.ts` 定义两种语言的页面元信息，`src/components/project-guide.tsx` 提供双语入门链接。两域名的 canonical 与 hreflang 统一指向 `.com` 对应页面；`.dev` 继续提供安装与下载。文档构建通过 `scripts/site_seo.py` 补充逐页元信息并生成全站 sitemap，排除跳转、打印和目录副本。

| 文件或目录 | 内容 |
| --- | --- |
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

随后补齐 `src/locales/en/messages.po` 中的新英文译文。生产页面的语言由 URL 确定，语言链接同时更新 URL、正文和元信息，并支持浏览器前进与后退。未预渲染的开发页面仍优先使用本地保存的 `cjv-lang`，再根据浏览器语言选择。生产构建由 Lingui Vite 插件自动编译词条目录。

视觉主题在 `src/style.css` 中定义，深色模式跟随系统设置，字体由本地依赖打包。界面动效需继续遵循 `useReducedMotion` 的无障碍偏好。

## 部署内容

在 CI 部署中，`.github/workflows/pages.yml` 会先构建 `web/` 前端，再向 `web/dist/` 中放入各平台的 `/dl/` `cjv-init` 安装器和两本双语手册。本地单独执行 Vite 构建不会生成这些额外目录。下载安装器来源见[发布流程](releasing.md#网页安装器)，文档构建输出见[文档站](documentation.md)。
