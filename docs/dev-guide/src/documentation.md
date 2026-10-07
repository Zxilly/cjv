# 文档站

`docs/user-guide/` 和 `docs/dev-guide/` 是两个独立 mdBook 项目。每本书的 `src/` 保存中文，`en/` 保存英文，共用 `book.toml` 和 `theme/`。

## 编辑章节

中英文源文件独立维护，内容变化应在同一提交中同步。新增、移动或删除章节时，同时修改两份 `SUMMARY.md`，保持两种语言的页面路径一致。代码中的命令、路径和标识符保持一致，注释可以翻译。

书内使用相对 `.md` 链接；跨书使用站点绝对 URL，因为两本书分别构建。移动已发布页面时，在 `book.toml` 的 `[output.html.redirect]` 中保留旧地址重定向。

`theme/cjv-i18n.js` 和 CSS 提供语言切换器。它替换 URL 中的 `/zh-CN/` 与 `/en/` 段，因此两种语言需要相同的页面路径。

## 本地预览

在对应书目录下执行：

```bash
mdbook serve --open
```

预览英文版（Bash）：

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook serve --open
```

PowerShell：

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
mdbook serve --open
```

结束英文预览后移除这两个环境变量，恢复默认中文配置。只构建时将 `serve --open` 换成 `build`。Markdown 检查命令见[测试与检查](testing.md)。

## Pages 部署

`.github/workflows/pages.yml` 在 master 推送、手动运行或发布流程调用时部署。它先构建 `web/`，再放入安装器和四份书：

```text
web/dist/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

顺序不能颠倒，`pnpm build` 会清空 `web/dist`。工作流使用 mdBook 0.5.3，并为每次构建设置源语言与 `site-url`。以用户手册英文版为例，在 `docs/user-guide/` 执行：

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/en/ mdbook build -d ../../web/dist/book/user-guide/en
```

整个 `web/dist` 作为一个 Pages artifact 部署。安装器来源与发布顺序见[发布流程](releasing.md)。
