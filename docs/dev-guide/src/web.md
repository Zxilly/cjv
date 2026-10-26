# 落地页

`web/` 是部署在 [cjv.zxilly.com](https://cjv.zxilly.com) 和 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的静态落地页，提供一键安装脚本命令和各平台二进制下载。它使用 React、Vite、TypeScript、Tailwind CSS 和 Lingui 构建；依赖及 `pnpm` 版本均固定在 `web/package.json` 中。

## 开发与文件位置

在 `web/` 目录下启动本地开发服务器：

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` 会先运行 TypeScript 类型检查，再生成 `web/dist/` 生产产物；`pnpm preview` 用于在本地预览构建结果。

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

修改平台识别逻辑后，请务必运行真实浏览器平台测试（`pnpm test:integration` 和 `pnpm test:built`），确认无法识别架构时不会错误推荐单一二进制。测试命令见[测试与检查](testing.md#安装脚本与前端)。

## 翻译与样式

中文（`zh`）是 Lingui 的源语言：

- JSX 元素使用 `@lingui/react/macro` 的 `<Trans>`。
- 组件内字符串表达式使用 `useLingui().t`。
- 组件外消息描述符使用 `@lingui/core/macro` 的 `msg`。

添加或修改前端文案后，提取词条：

```bash
pnpm i18n:extract
```

随后补齐 `src/locales/en/messages.po` 中的新英文译文。页面在运行时优先使用本地保存的 `cjv-lang`，未保存时根据浏览器语言选择中文或英文；生产构建由 Lingui Vite 插件自动编译词条目录。

视觉主题在 `src/style.css` 中定义，深色模式跟随系统设置，字体由本地依赖打包。界面动效需继续遵循 `useReducedMotion` 的无障碍偏好。

## 部署内容

在 CI 部署中，`.github/workflows/pages.yml` 会先构建 `web/` 前端，再向 `web/dist/` 中放入各平台的 `/dl/` `cjv-init` 安装器和两本双语手册。本地单独执行 Vite 构建不会生成这些额外目录。下载安装器来源见[发布流程](releasing.md#网页安装器)，文档构建输出见[文档站](documentation.md)。
