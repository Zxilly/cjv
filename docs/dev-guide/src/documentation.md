# 文档站

`docs/user-guide/` 和 `docs/dev-guide/` 是两个独立的 mdBook 项目。每本书的 `src/` 目录保存简体中文源文件，`en/` 目录保存英文源文件，共用同一份 `book.toml` 配置；两本书的站点主题与语言切换器统一保存在 `docs/theme/`。

## 编辑章节

中英文源文件独立维护，内容变化应在同一提交中同步完成：

- 新增、移动或删除章节时，请同时修改 `src/SUMMARY.md` 和 `en/SUMMARY.md`，保持两种语言的页面文件路径完全一致。
- 代码块中的命令、路径、配置键和标识符保持一致，正文与注释可以按语言翻译。
- 同一本书内部使用相对 `.md` 链接；跨书（用户手册与开发指南之间）使用站点绝对 URL（`https://cjv.zxilly.com/book/...`），因为两本书分别构建。
- 移动或重命名已发布页面时，请在 `book.toml` 的 `[output.html.redirect]` 中保留旧地址重定向。

`docs/theme/cjv-i18n.js` 和 `cjv-i18n.css` 提供顶栏语言切换器。它通过替换当前 URL 中的 `/zh-CN/` 与 `/en/` 路径段来切换语言，因此两种语言必须具有相同的页面路径。由于中英文标题生成的 HTML 锚点 ID 不同，切换语言会打开对应章节顶部，并去掉原语言的 `#anchor` 标题锚点。

## 写作约定

- 使用客观、直接的技术说明，写明命令行为、前提条件、默认值和限制。
- README 保留项目名称、用途、安装和快速上手；详细选项放在用户手册或命令参考中。
- 标题使用简短的主题名称或操作名称。正文避免宣传性评价、重复的章节引导和装饰性加粗链接。
- 行为描述以代码和测试为依据，区分默认行为、显式选项和平台差异；中英文说明保持一致。

## 本地预览

### 完整站点构建与检查（推荐）

安装 **mdBook 0.5.3** 和 **Python 3.9 以上版本**，在仓库根目录执行：

```bash
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist 8000
```

在浏览器中访问 `http://localhost:8000/book/user-guide/zh-CN/`（或 `/en/`），即可预览与线上发布完全一致的主题和语言切换器。在 Windows 上如果未配置 `python3`，可将其换为 `python`。

`scripts/build-docs.py` 会为四份书（用户手册与开发指南的中英文版）设置源语言、标题和站点路径，在临时目录中加入共享主题，并自动检查所有生成的 HTML 页面中的内部链接、跨书 `/book/` 链接、静态资源和 `#anchor` 锚点。传入 `--output <dir>` 可指定其他输出目录，`--check-only` 只检查已有产物而不重新调用 `mdbook`。Markdown 格式检查命令见[测试与检查](testing.md#文档工作流和依赖)。

### 预览单本书（`mdbook serve`）

如果只想在编辑时实时预览单本书的内容，可在 `docs/user-guide/` 或 `docs/dev-guide/` 目录下直接执行（注意此入口使用 mdBook 默认主题，不含共享语言切换器）：

```bash
mdbook serve --open
```

在 Bash 中预览英文版：

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_BOOK__TITLE="cjv User Guide" mdbook serve --open
```

在 Windows PowerShell 中预览英文版：

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
$env:MDBOOK_BOOK__TITLE = "cjv User Guide"
mdbook serve --open
```

预览开发指南英文版时使用标题 `"cjv Developer Guide"`。结束英文预览后请移除这三个环境变量以恢复默认中文配置；如果只需构建而不启动预览服务，将 `serve --open` 换成 `build` 即可。

## Pages 部署

`.github/workflows/pages.yml` 在相关前端、文档或部署文件推送到 `master`、手动运行或被发布流程调用时部署。它先构建 `web/` 前端，再向 `web/dist/` 中放入安装器和四份手册输出：

```text
web/dist/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

请保持这一构建顺序：在 `web/` 中执行 `pnpm build` 会清空 `web/dist/`。PR 检查和发布工作流使用完全相同的 mdBook 版本与构建入口：

```bash
python3 scripts/build-docs.py
```

整个 `web/dist` 作为一个 `github-pages` artifact 部署到 GitHub Pages 与腾讯云 EdgeOne。安装器来源与发布顺序见[发布流程](releasing.md)。
