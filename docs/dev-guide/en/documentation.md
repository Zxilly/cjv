# Documentation site

cjv's documentation consists of two mdBooks: the user guide in `docs/user-guide/` and the developer guide (the one you are reading) in `docs/dev-guide/`. The two books have the same structure, and each keeps two independent sources: `src/` is Simplified Chinese (the source language) and `en/` is English.

The landing page (`web/`, React + Vite) is a separate subsystem covered in its own chapter; see [web.md](web.md). This chapter covers only the two books.

## Directory layout

Each book is a standalone mdBook project laid out like this:

```text
docs/user-guide/
├── book.toml         # mdBook configuration
├── theme/            # language switcher (the globe dropdown in the top right)
│   ├── cjv-i18n.js
│   └── cjv-i18n.css
├── src/              # Simplified Chinese source (.md + SUMMARY.md)
│   ├── SUMMARY.md
│   ├── introduction.md
│   └── ...
└── en/               # English source, mirroring the structure of src/
    ├── SUMMARY.md
    ├── introduction.md
    └── ...
```

`docs/dev-guide/` is identical. `book/` (the local build output) is not committed; see `docs/*/book/` in `.gitignore`.

The two sources are independent Markdown files with no automatic translation: Chinese goes in `src/`, English goes in `en/`, and the two keep the same chapter filenames and SUMMARY structure. This lets code blocks be localized independently (Chinese comments in the Chinese version, English in the English one), at the cost of having to update the English by hand whenever the Chinese changes; see below.

## book.toml

The `book.toml` of the two books is nearly identical. The key entries:

```toml
[book]
language = "zh-CN"          # source language; overridden with MDBOOK_BOOK__LANGUAGE=en for the English build
src = "src"                 # default source dir; overridden with MDBOOK_BOOK__SRC=en for the English build

[output.html]
site-url = "/"              # default; CI overrides it per language via MDBOOK_OUTPUT__HTML__SITE_URL
git-repository-url = "https://github.com/Zxilly/cjv"
edit-url-template = "https://github.com/Zxilly/cjv/edit/master/docs/user-guide/{path}"
additional-js = ["theme/cjv-i18n.js"]
additional-css = ["theme/cjv-i18n.css"]
```

One `book.toml` builds both languages: Chinese with a plain `mdbook build` (using `src/` and `language=zh-CN`); English with `MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook build`, which switches the source dir to `en/` and sets `<html lang>` to en. mdBook flattens nested TOML keys into environment variables with double underscores: `MDBOOK_BOOK__SRC` maps to `src` under `[book]`, and `MDBOOK_BOOK__LANGUAGE` maps to `language` under `[book]`.

## Language switcher

The globe dropdown in the top right is injected by `theme/cjv-i18n.js`, styled by `theme/cjv-i18n.css`, and wired in through the `additional-js` / `additional-css` entries in `book.toml`. The two language builds produce identical page paths (because the SUMMARY structure is the same), so switching languages just rewrites the `/zh-CN/` or `/en/` language segment of the current URL to the other one and navigates there. To add a language later, edit the `LANGS` array at the top of `cjv-i18n.js`.

## Adding or editing chapters

To edit a chapter's body, edit the corresponding `.md` under `src/`, and update the same-named file under `en/` to match. To add a chapter: create a same-named Markdown file under both `src/` and `en/`, then register it in both `SUMMARY.md` files, otherwise mdBook will not include it in the table of contents. `SUMMARY.md` is a list tree; indentation expresses depth and group titles use `#`:

```markdown
# Getting started

- [Building from source](building.md)
- [Code architecture](architecture.md)
```

Chapters reference each other with relative links by filename in the same directory, such as `[Testing](testing.md)`. To point at the user guide, use the absolute site link <https://cjv.zxilly.dev/book/user-guide/en/> or a GitHub source link. Do not write relative paths across books, since the two books are built and deployed separately.

## Keeping the two languages in sync

The two languages are independent sources. There is no PO file or fuzzy matching to flag what fell behind, so when you change the Chinese you must remember to change the English, otherwise the English site stays on the old content. The convention is to put the Chinese source and the corresponding `en/` changes in the same PR so a reviewer can read them side by side.

Code blocks are written separately in the two sources: the Chinese version in `src/` may use Chinese in its comments and example output, while the English version in `en/` uses English. Commands, paths, and identifiers stay identical on both sides.

## Building and previewing locally

While writing, use `mdbook serve` to view the Chinese source; it starts a local server and rebuilds on every change:

```bash
cd docs/user-guide
mdbook serve --open
```

`serve` renders Chinese by default, following `language = "zh-CN"` and `src = "src"` in `book.toml`. To preview the English version, switch the source dir to `en/`:

```bash
cd docs/user-guide
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook serve --open
```

## CI deployment

The documentation site is not deployed on its own; it ships together with the landing page through the `Pages` workflow (`.github/workflows/pages.yml`), triggered on a push to `master`, with the output uploaded to GitHub Pages. The step order matters: the workflow first runs `pnpm build` to build the landing page into `web/dist`, then builds the two books into `web/dist/book/`, because `pnpm build` clears `web/dist`.

Before building the mdBooks it installs `mdbook` (pinned, currently `0.5.3`). Each book is built twice, once Chinese and once English, using environment overrides to switch the source dir and language:

```bash
# The dev-guide step; user-guide works the same way
cd docs/dev-guide
MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/zh-CN/ mdbook build -d ../../web/dist/book/dev-guide/zh-CN
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/dev-guide/en/ mdbook build -d ../../web/dist/book/dev-guide/en
```

The four outputs of the two books end up at:

```text
web/dist/book/
├── user-guide/{en,zh-CN}/
└── dev-guide/{en,zh-CN}/
```

Each output's `site-url` is set to its own subpath so that in-site links, the search index, and asset references resolve correctly under that subpath. The whole `web/dist` (landing page plus the four book outputs) is uploaded and deployed as a single Pages artifact. The final online path is `https://cjv.zxilly.dev/book/<book>/<lang>/`; for example, the Chinese version of this book is at `/book/dev-guide/zh-CN/`.

For the full CI picture, see [ci.md](ci.md).
