# 落地页

`web/` 目录是部署在 [cjv.zxilly.com](https://cjv.zxilly.com) 和 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的静态落地页，用于向访问者提供平台对应的安装脚本命令和独立安装器下载。该项目使用 React、Vite、TypeScript、Tailwind CSS 和 Lingui 构建，依赖版本与 `pnpm` 版本均固定在 `web/package.json` 中。

## 本地开发与文件结构

在 `web/` 目录下执行以下命令启动本地开发服务器：

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

- `pnpm build`：先执行 TypeScript 类型检查，再将静态产物构建到 `web/dist/`。
- `pnpm preview`：在本地预览已构建的 `web/dist/` 目录。

| 文件或目录 | 用途说明 |
| --- | --- |
| `src/App.tsx` | 安装落地页主体布局、镜像源切换与下载交互 |
| `src/hooks/use-platform.ts` | 浏览器操作系统与 CPU 架构识别及下载建议 |
| `src/generated/platforms.ts` | 由 Go 平台清单（`internal/target/catalog.go`）生成的前端数据 |
| `src/lib/i18n.ts`、`src/locales/` | 语言选择、持久化状态与 Lingui `.po` 翻译文件 |
| `src/style.css`、`src/components/` | Tailwind CSS 主题变量与界面组件 |
| `public/install.sh`、`public/install.ps1` | 公开分发的 Unix 与 PowerShell 安装脚本 |

> **注意：** 请勿直接手动编辑 `web/src/generated/platforms.ts`。如需修改支持的平台，请更新 Go 的 `internal/target/catalog.go` 并在仓库根目录运行 `go generate ./...`（见[从源码构建](building.md#交叉编译与生成文件同步)）。

## 浏览器平台检测规则

### 检测状态与 macOS 架构处理

`use-platform.ts` 先使用同步浏览器信息（`navigator.userAgent` 与 `navigator.platform`）计算初始结果，再尝试通过 UA Client Hints（`navigator.userAgentData`）获取更精确的 CPU 架构信息。检测结果分为三种状态：

- `ready`：操作系统与 CPU 架构均已确认且受支持。
- `unknown`：操作系统受支持，但浏览器无法可靠确认具体的 CPU 架构。
- `unsupported`：移动操作系统或没有预编译二进制文件的架构。

在 macOS 上，部分浏览器即使运行在 Apple Silicon（`arm64`）设备上仍会报告 Intel（`x86_64`）User-Agent。当无法从浏览器可靠确认 Mac 架构时，页面仍会提供 `install.sh` 一键安装脚本（脚本会在终端本机检测真实架构），并在手动下载面板中同时列出 Apple Silicon 和 Intel 两个选项。检测逻辑还专门处理了 iPadOS 桌面模式的 User-Agent 差异。

### 鸿蒙（HarmonyOS / OpenHarmony）检测规则

针对鸿蒙访问者，检测器遵循以下严格规则：

1. **系统版本优先级**：优先读取 `OpenHarmony` 或 `HarmonyOS` 标记中的版本号并保留其版本命名空间，绝不将 `ArkWeb`、`HuaweiBrowser` 或 `Chrome` 的版本号误判为操作系统版本。
2. **设备形态区分**：
   - `Phone` 或 `Mobile` 识别为手机（`unsupported`）。
   - `Tablet` 同时包含 `Windows NT` 时，识别为处于平板模式的二合一设备（`ready` 或 `unknown`）。
   - `PC` 或桌面 `Windows NT` 标记识别为鸿蒙电脑（`ready` 或 `unknown`）。
   - 单独的 `Tablet` 识别为普通平板（`unsupported`）。
3. **架构安全边界**：兼容标记（如 `Windows NT`、`Linux`、`Android`）及 UA Client Hints 不会将鸿蒙覆盖识别为 Windows 或 Android，也不用于猜测鸿蒙设备的原生 CPU 架构。下载面板会引导用户在终端运行 `uname -m` 确认 `ARM64` 或 `x86_64` 架构，并在终端执行 `cjv-init`。

修改平台识别逻辑后，请务必运行真实浏览器平台测试（`pnpm test:integration` 和 `pnpm test:built`），确认在架构未知时不会错误推荐单一架构的二进制文件。测试命令见[测试与检查](testing.md#安装脚本与前端测试)。

## 多语言翻译与样式

中文（`zh`）是 Lingui 的源语言：

- JSX 元素使用 `@lingui/react/macro` 的 `<Trans>`。
- 组件内的字符串表达式使用 `useLingui().t`。
- 组件外的消息描述符使用 `@lingui/core/macro` 的 `msg`。

新增或修改前端文案后，提取词条目录：

```bash
pnpm i18n:extract
```

随后补齐 `web/src/locales/en/messages.po` 中的英文翻译。页面在运行时优先读取本地保存的 `cjv-lang` 设置，未保存时根据浏览器语言自动选择中文或英文。生产构建时由 Lingui Vite 插件自动编译词条目录。

主题色与排版在 `src/style.css` 中定义，深色模式跟随系统设置，字体通过本地依赖包打包。所有界面动效均需继续遵循 `useReducedMotion` 的用户无障碍偏好。

## 部署产物组成

在 `web/` 中执行本地 `pnpm build` 只会生成落地页静态文件和 `install.sh` / `install.ps1`。正式部署时，`.github/workflows/pages.yml` 会在前端构建完成后向 `web/dist/` 补充两部分内容：

- 各平台的独立安装器 `/dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]`，来源见[发布流程](releasing.md#网页独立安装器dl)。
- 用户手册与开发指南的双语文档站 `/book/user-guide/{zh-CN,en}/` 与 `/book/dev-guide/{zh-CN,en}/`，构建方式见[文档站](documentation.md)。
