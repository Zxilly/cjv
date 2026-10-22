# Documentation site

`docs/user-guide/` and `docs/dev-guide/` are independent mdBook projects. Each keeps Chinese sources in `src/` and English in `en/`, sharing `book.toml`. Both books use the site theme in `docs/theme/`.

## Edit chapters

The languages are maintained separately. Synchronize content changes in the same commit. When adding, moving, or deleting chapters, update both `SUMMARY.md` files and keep page paths identical across languages. Commands, paths, and identifiers stay the same; comments may be translated.

Use relative `.md` links within a book and absolute site URLs between books, because the books build separately. When moving a published page, retain its old URL in `[output.html.redirect]` in `book.toml`.

`docs/theme/cjv-i18n.js` and its CSS provide the language switcher. It replaces the `/zh-CN/` and `/en/` URL segments, so both languages need matching page paths. Switching opens the corresponding chapter at the top and removes the previous language's heading anchor. The URL determines the current language.

## Local preview

Install mdBook 0.5.3 and Python 3.9 or later, then run from the repository root:

```bash
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist 8000
```

Open `http://localhost:8000/book/user-guide/en/` to preview the published theme and language switcher. On Windows, use `python` if `python3` is unavailable. The script sets the source language, title, and site path for all four books, stages the shared theme in a temporary directory, and checks internal pages, assets, and anchors. Use `--output <dir>` for another output directory or `--check-only` to check an existing build.

To preview one book's content, run from its directory:

```bash
mdbook serve --open
```

This entry point uses the default mdBook theme. For English in Bash:

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_BOOK__TITLE="cjv User Guide" mdbook serve --open
```

In PowerShell:

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
$env:MDBOOK_BOOK__TITLE = "cjv User Guide"
mdbook serve --open
```

Use `cjv Developer Guide` for the developer book. Remove these three environment variables after the English preview to restore the Chinese defaults. Replace `serve --open` with `build` to build without serving. Markdown checks are listed in [testing and checks](testing.md).

## Pages deployment

`.github/workflows/pages.yml` deploys when relevant files are pushed to master, on manual runs, or on release-workflow calls. It builds `web/` first, then adds installers and four book outputs:

```text
web/dist/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

Keep this order: `pnpm build` clears `web/dist`. Pull request checks and deployments use the same mdBook version and build entry point:

```bash
python3 scripts/build-docs.py
```

The entire `web/dist` is deployed as one Pages artifact. See [releases](releasing.md) for installer artifacts and release ordering.
