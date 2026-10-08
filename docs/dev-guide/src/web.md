# 落地页

`web/` 是部署在 [cjv.zxilly.dev](https://cjv.zxilly.dev) 的静态页面，提供安装命令和平台下载。它使用 React、Vite、TypeScript、Tailwind CSS 和 Lingui；依赖及 pnpm 版本以 `web/package.json` 为准。

## 开发与文件位置

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` 先用 TypeScript 检查类型，再生成 `dist/`。`pnpm preview` 预览构建产物。

| 文件或目录 | 内容 |
| --- | --- |
| `src/App.tsx` | 安装页面与交互 |
| `src/hooks/use-platform.ts` | 浏览器平台识别与下载建议 |
| `src/generated/platforms.ts` | Go 平台清单生成的前端数据 |
| `src/lib/i18n.ts`、`src/locales/` | 语言选择与翻译 |
| `src/style.css`、`src/components/` | 主题和组件 |
| `public/install.sh`、`public/install.ps1` | 安装脚本 |

修改支持平台时，更新 Go 的 `internal/target` 清单并运行 `go generate ./...`，不要直接编辑生成的 TypeScript 文件。

## 平台检测

页面先用同步浏览器信息计算结果，再尝试 UA Client Hints 补充架构信息。结果分为 `ready`、`unsupported` 和 `unknown`。

macOS 浏览器无法可靠给出 CPU 架构时，安装脚本仍可用，但手动下载显示 Apple Silicon 与 Intel 两个选项。移动系统及没有预编译二进制的架构显示不支持。检测还处理 iPadOS 桌面模式和 HarmonyOS NEXT 的 UA 差异。

鸿蒙检测优先读取 `OpenHarmony` / `HarmonyOS` 标记中的版本，保留其版本名称，不将 `ArkWeb`、`HuaweiBrowser` 或 Chrome 版本当作系统版本。`Phone` / `Mobile` 表示手机，`Tablet` 表示平板；根据已收集的 UA 样例，`Tablet` 同时包含 `Windows NT` 时显示二合一设备平板模式的提示。这些兼容标记及 UA Client Hints 不会将鸿蒙识别为 Windows 或 Android，也不用于推断鸿蒙设备的原生架构。目前鸿蒙仍显示不支持，不推荐安装包；后续适配入口留有 TODO。

改动此处时运行真实浏览器平台测试，确认无法识别架构时不会错误推荐单一二进制。测试命令见[测试与检查](testing.md)。

## 翻译与样式

中文是 Lingui 源语言。JSX 使用 `@lingui/react/macro` 的 `<Trans>`，组件内字符串使用 `useLingui().t`，组件外描述符使用 `@lingui/core/macro` 的 `msg`。添加文案后执行：

```bash
pnpm i18n:extract
```

补齐 `src/locales/en/messages.po` 中的新译文。运行时优先使用保存的 `cjv-lang`，再根据浏览器语言选择中文或英文。生产构建由 Lingui Vite 插件处理 catalog。

主题在 CSS 中定义，深色模式跟随系统，字体由本地依赖打包。动效应继续遵循 `useReducedMotion` 的用户偏好。

## 部署内容

Pages 在前端构建后加入各平台的 `cjv-init` 和两本双语手册。Vite 本地构建本身不生成这些文件。下载链接来源见[发布流程](releasing.md)，文档输出见[文档站](documentation.md)。
