# Documentation site

`docs/user-guide/` and `docs/dev-guide/` are independent mdBook projects. Each keeps Chinese sources in `src/` and English in `en/`, sharing `book.toml` and `theme/`.

## Edit chapters

The languages are maintained separately. Synchronize content changes in the same commit. When adding, moving, or deleting chapters, update both `SUMMARY.md` files and keep page paths identical across languages. Commands, paths, and identifiers stay the same; comments may be translated.

Use relative `.md` links within a book and absolute site URLs between books, because the books build separately. When moving a published page, retain its old URL in `[output.html.redirect]` in `book.toml`.

`theme/cjv-i18n.js` and its CSS provide the language switcher. It replaces the `/zh-CN/` and `/en/` URL segments, so both languages need matching page paths.

## Local preview

From the book directory:

```bash
mdbook serve --open
```

For English in Bash:

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook serve --open
```

In PowerShell:

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
mdbook serve --open
```

Remove these environment variables after the English preview to restore the Chinese defaults. Replace `serve --open` with `build` to build without serving. Markdown checks are listed in [testing and checks](testing.md).

## Pages deployment

`.github/workflows/pages.yml` deploys on master pushes, manual runs, or release-workflow calls. It builds `web/` first, then adds installers and four book outputs:

```text
web/dist/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

Keep this order: `pnpm build` clears `web/dist`. The workflow uses mdBook 0.5.3 and sets the source language and `site-url` for each build. For example, build the English user guide from `docs/user-guide/`:

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_OUTPUT__HTML__SITE_URL=/book/user-guide/en/ mdbook build -d ../../web/dist/book/user-guide/en
```

The entire `web/dist` is deployed as one Pages artifact. See [releases](releasing.md) for installer artifacts and release ordering.
