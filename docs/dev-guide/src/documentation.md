# 文档站

cjv 的文档由两本 mdBook 组成：用户手册在 `docs/user-guide/`，开发指南(你正在读的这本)在 `docs/dev-guide/`。两本书结构完全一样，每本都维护中英两份独立的源：`src/` 是简体中文(源语言)，`en/` 是英文。

落地页(`web/`，React + Vite)是另一个子系统，单独一章讲，见 [web.md](web.md)。本章只讲两本书。

## 目录布局

每本书是一个独立的 mdBook 项目，目录长这样：

```text
docs/user-guide/
├── book.toml         # mdBook 配置
├── theme/            # 语言切换器(右上角地球图标下拉)
│   ├── cjv-i18n.js
│   └── cjv-i18n.css
├── src/              # 简体中文源(.md + SUMMARY.md)
│   ├── SUMMARY.md
│   ├── introduction.md
│   └── ...
└── en/               # 英文源,结构与 src/ 镜像
    ├── SUMMARY.md
    ├── introduction.md
    └── ...
```

`docs/dev-guide/` 一模一样。`book/`(本地构建产物)不入库，见 `.gitignore` 的 `docs/*/book/`。

中英两份源是各自独立的 Markdown 文件，没有自动翻译机制：中文写在 `src/`，英文写在 `en/`，两边章节文件名和 SUMMARY 结构保持一致。这样代码块能各自本地化(中文版注释用中文、英文版用英文)，代价是改了中文得手动同步英文，见下文。

## book.toml

两本书的 `book.toml` 几乎相同，关键几项：

```toml
[book]
language = "zh-CN"          # 源语言;CI 构建英文版时用 MDBOOK_BOOK__LANGUAGE=en 覆盖
src = "src"                 # 默认源目录;CI 构建英文版时用 MDBOOK_BOOK__SRC=en 覆盖

[output.html]
site-url = "/"              # 默认值;CI 逐语言用 MDBOOK_OUTPUT__HTML__SITE_URL 覆盖
git-repository-url = "https://github.com/Zxilly/cjv"
edit-url-template = "https://github.com/Zxilly/cjv/edit/master/docs/user-guide/{path}"
additional-js = ["theme/cjv-i18n.js"]
additional-css = ["theme/cjv-i18n.css"]
```

一份 `book.toml` 构建两种语言：中文直接 `mdbook build`(用 `src/`、`language=zh-CN`)；英文用 `MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook build`，把源目录切到 `en/`、把 `<html lang>` 设成 en。mdBook 用双下划线把 TOML 嵌套键拍平成环境变量，`MDBOOK_BOOK__SRC` 对应 `[book]` 下的 `src`，`MDBOOK_BOOK__LANGUAGE` 对应 `[book]` 下的 `language`。

## 语言切换器

右上角那个地球图标下拉由 `theme/cjv-i18n.js` 注入，样式在 `theme/cjv-i18n.css`，经 `book.toml` 的 `additional-js` / `additional-css` 挂上。两种语言构建出来的页面路径一致(因为 SUMMARY 结构相同)，所以切换语言只是把当前 URL 里的 `/zh-CN/` 或 `/en/` 语言段替换成另一个再跳转。将来要加语言，改 `cjv-i18n.js` 顶部的 `LANGS` 数组即可。

## 新增或编辑章节

改一章正文，直接编辑 `src/` 下对应的 `.md`，英文同步改 `en/` 下同名文件。新增一章：在 `src/` 和 `en/` 各建一个同名 Markdown 文件，再在两边的 `SUMMARY.md` 里登记，否则 mdBook 不会把它纳入目录。`SUMMARY.md` 是一棵列表树，缩进表示层级，分组标题用 `#`：

```markdown
# 上手开发

- [从源码构建](building.md)
- [代码架构](architecture.md)
```

章节之间互相引用，同目录下用文件名相对链接，比如 `[测试](testing.md)`。要指向用户手册，用线上站点的绝对链接 <https://cjv.zxilly.dev/book/user-guide/zh-CN/>，或者直接给 GitHub 源链。不要写跨书的相对路径，两本书是分开构建分开部署的。

## 中英同步

中英是两份独立源，没有 PO、没有 fuzzy 匹配那套机制提醒你哪里没跟上，改了中文要记得手动改英文，否则英文站会停在旧内容。约定是：中文源和对应的 `en/` 改动放在同一个 PR 里，reviewer 能对照着看。

代码块在两份源里各写各的：中文版 `src/` 的代码注释、示例输出可以用中文，英文版 `en/` 用英文。命令、路径、标识符两边保持一致。

## 本地构建与预览

写作时用 `mdbook serve` 看中文源，它会起本地服务器并随改随重建：

```bash
cd docs/user-guide
mdbook serve --open
```

`serve` 默认按 `book.toml` 里的 `language = "zh-CN"` 和 `src = "src"` 渲染中文。要预览英文版，把源目录切到 `en/`：

```bash
cd docs/user-guide
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook serve --open
```

## CI 部署

文档站不单独部署，它搭着落地页一起走 `Pages` 工作流(`.github/workflows/pages.yml`)，推到 `master` 时触发，产物上传到 GitHub Pages。步骤顺序很关键：工作流先 `pnpm build` 构建落地页到 `web/dist`，再把两本书构建进 `web/dist/book/` 下，因为 `pnpm build` 会清空 `web/dist`。

构建 mdBook 前先装 `mdbook`(固定版本，目前 `0.5.3`)。每本书构建两遍，一遍中文一遍英文，靠环境变量覆盖切源目录和语言：

```bash
# 开发指南那一步,user-guide 同理
cd docs/dev-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/zh-CN/ mdbook build -d ../../web/dist/book/dev-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/en/ mdbook build -d ../../web/dist/book/dev-guide/en
```

两本书四份产物最终落在：

```text
web/dist/book/
├── user-guide/{en,zh-CN}/
└── dev-guide/{en,zh-CN}/
```

每份产物的 `site-url` 设成它自己的子路径，站内链接、搜索索引、资源引用才能在子路径下正确解析。整个 `web/dist`(落地页 + 四份书)作为一个 Pages artifact 上传部署。最终线上路径就是 `https://cjv.zxilly.dev/book/<book>/<lang>/`，比如本书中文版在 `/book/dev-guide/zh-CN/`。

CI 流程的全貌见 [ci.md](ci.md)。
