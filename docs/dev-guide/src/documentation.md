# 文档站

`docs/user-guide/`（用户手册）和 `docs/dev-guide/`（开发指南）是两个独立的 mdBook 项目：

- 每本书的 `src/` 目录存放简体中文源文件，`en/` 目录存放英文源文件，共用同一个 `book.toml` 配置文件。
- 两本书共用保存在 `docs/theme/` 下的站点主题与语言切换器（`cjv-i18n.js` 和 `cjv-i18n.css`）。

本章介绍如何编辑并同步双语章节、如何在本地构建和预览文档站，以及 Pages 部署的目录组装顺序。

## 编辑章节与双语同步

### 保持中英文结构与路径一致

中英文源文件独立维护，内容变更应在同一提交中同步完成：

- 新增、重命名、移动或删除章节时，请同时修改 `src/SUMMARY.md` 和 `en/SUMMARY.md`，确保两种语言下每个章节的文件相对路径完全一致。
- 代码块中的命令、文件路径、配置键名和标识符保持一致，注释文字可以按语言翻译。
- 移动或删除已发布的页面时，请在对应 `book.toml` 的 `[output.html.redirect]` 中添加旧地址重定向规则，防止外部链接失效。

### 链接与语言切换规则

- **同一本书内部跳转**：使用相对 `.md` 链接，例如 `[项目工具链](toolchain-file.md)` 或 `[安装 cjv](../installation/index.md)`。
- **跨书跳转（用户手册与开发指南之间）**：使用站点绝对 URL（如 `https://cjv.zxilly.com/book/user-guide/zh-CN/...` 或 `https://cjv.zxilly.com/book/dev-guide/zh-CN/...`），因为两本书是分别独立构建的。`scripts/build-docs.py` 会在本地构建完成后自动校验指向 `cjv.zxilly.com` 和 `cjv.zxilly.dev` 的所有 `/book/...` 跨书链接。
- **语言切换器原理**：`docs/theme/cjv-i18n.js` 通过将当前 URL 路径中的 `/zh-CN/` 与 `/en/` 互相替换来切换语言。由于中英文标题生成的 HTML 锚点 ID 不同，切换语言时会直接打开对应语言章节的顶部，并去掉原语言的 `#anchor` 片段。

## 本地构建与预览

### 完整站点构建与链接检查（推荐）

安装 **mdBook 0.5.3** 和 **Python 3.9+** 后，在仓库根目录执行：

```bash
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist 8000
```

在浏览器中访问 `http://localhost:8000/book/user-guide/zh-CN/`（或 `/en/`），即可预览与线上发布完全一致的主题和语言切换器。在 Windows 上如果未配置 `python3` 命令，请改用 `python`。

`scripts/build-docs.py` 会依次完成以下工作：

1. 将每本书的源文件、`book.toml` 以及共享主题 `docs/theme/` 复制到临时工作目录。
2. 为四份手册（用户手册与开发指南的中英文版）分别设置源目录、语言标识、书名标题和站点路径，调用 `mdbook build` 输出到 `web/dist/book/`。
3. 解析所有生成的 HTML 页面，检查全部内部页面链接、跨书 `/book/` 链接、静态资源和 `#anchor` 标题锚点是否有效。

传入 `--output <dir>` 可指定其他输出目录，传入 `--check-only` 可只检查已有构建产物。提交前请同时运行 `npx --yes markdownlint-cli2@0.22.1` 检查 Markdown 规范（见[测试与检查](testing.md#文档工作流与依赖检查)）。

### 单本书实时预览（`mdbook serve`）

如果只想在编写时实时预览单本书的内容，可以进入 `docs/user-guide/` 或 `docs/dev-guide/` 目录运行 `mdbook serve`。注意此入口使用 mdBook 默认主题，不包含 `docs/theme/` 语言切换器。

预览默认中文版：

```bash
mdbook serve --open
```

在 Bash 中预览英文版：

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_BOOK__TITLE="cjv User Guide" mdbook serve --open
```

在 PowerShell 中预览英文版：

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
$env:MDBOOK_BOOK__TITLE = "cjv User Guide"
mdbook serve --open
```

预览开发指南英文版时，将标题设为 `"cjv Developer Guide"`。结束英文预览后，请清除这三个环境变量以恢复默认中文配置。

## Pages 部署与产物顺序

`.github/workflows/pages.yml` 在相关文档、前端或部署文件推送到 `master`、手动触发或被发布工作流调用时执行。它按以下固定顺序组装 `web/dist/`：

```text
web/dist/
  index.html, install.sh, install.ps1, assets/...   # 1. 在 web/ 中执行 pnpm build 生成
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]   # 2. 从发布归档中提取独立安装器
  book/user-guide/{zh-CN,en}/                       # 3. 执行 python3 scripts/build-docs.py 生成
  book/dev-guide/{zh-CN,en}/                        # 4. 执行 python3 scripts/build-docs.py 生成
```

> **注意：** 顺序绝不能颠倒——必须先在 `web/` 中执行 `pnpm build`，再放入 `/dl/` 安装器和构建四份文档，因为 `pnpm build` 会清空整个 `web/dist/` 目录。

PR 检查和线上发布工作流使用完全相同的 mdBook 版本和构建脚本入口（`python3 scripts/build-docs.py`）。组装完成的 `web/dist/` 会打包为单个 Pages 制品，同时部署到 GitHub Pages 和腾讯云 EdgeOne。安装器来源与发布顺序见[发布流程](releasing.md)，边缘函数与域名路由见仓库根目录下的 `docs/website-deployment.md`。
