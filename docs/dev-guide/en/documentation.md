# Documentation site

`docs/user-guide/` and `docs/dev-guide/` are two independent mdBook projects:

- Each book stores its Chinese source chapters in `src/` and its English source chapters in `en/`, sharing a single `book.toml` configuration file.
- Both books share the custom language-switcher theme in `docs/theme/` (`cjv-i18n.js` and `cjv-i18n.css`).

This chapter explains how to edit bilingual chapters, preview the books locally, validate links, and deploy to GitHub Pages and EdgeOne.

## Edit and synchronize chapters

### Keep Chinese and English in sync

Maintain the Chinese (`src/`) and English (`en/`) files in the same commit:

- When you add, rename, move, or delete a chapter, update both `src/SUMMARY.md` and `en/SUMMARY.md` so every page has the exact same relative file path in both languages.
- Keep CLI commands, file paths, configuration keys, and code identifiers identical across both languages; translate prose and code comments only.
- When you move or delete a published chapter, add a redirect entry under `[output.html.redirect]` in that book's `book.toml` file so existing links do not break.

### Cross-reference and link rules

- **Within the same book**: use relative Markdown links such as `[Project toolchains](toolchain-file.md)` or `[Install cjv](../installation/index.md)`.
- **Between User Guide and Developer Guide**: use full public URLs (`https://cjv.zxilly.com/book/user-guide/en/...` or `https://cjv.zxilly.com/book/dev-guide/en/...`) because each mdBook builds separately. `scripts/build-docs.py` resolves and validates all `/book/...` links on `cjv.zxilly.com` and `cjv.zxilly.dev` against the local build output.
- **Language switcher behavior**: `docs/theme/cjv-i18n.js` switches languages by replacing `/zh-CN/` with `/en/` (or vice versa) in the current URL path. Because translated headings generate different HTML anchor IDs, switching languages opens the top of the matching chapter and strips the previous heading hash (`#...`).

## Build and preview locally

### Full site build and link check (recommended)

Install **mdBook 0.5.3** and **Python 3.9+**, then run these commands from the repository root:

```bash
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist 8000
```

Open `http://localhost:8000/book/user-guide/en/` (or `/zh-CN/`) in your browser. On Windows, use `python` if `python3` is not defined.

`scripts/build-docs.py` performs these steps:

1. Copies each book (`user-guide` and `dev-guide`) and the shared `docs/theme/` assets into a temporary staging directory.
2. Runs `mdbook build` for all four books (`zh-CN` and `en` for both guides) with the proper `MDBOOK_BOOK__SRC`, `MDBOOK_BOOK__LANGUAGE`, `MDBOOK_BOOK__TITLE`, and `MDBOOK_OUTPUT__HTML__SITE_URL` environment variables.
3. Parses every generated HTML page and checks that all local links, cross-book `/book/` URLs, static assets, and `#anchor` fragments exist.

Pass `--output <dir>` to write the built books to a custom directory, or pass `--check-only` to validate an existing build directory without rebuilding. Also run `npx --yes markdownlint-cli2@0.22.1` to check Markdown style; read [Tests and checks](testing.md#documentation-workflows-and-dependencies).

### Single-book live preview (`mdbook serve`)

If you want live reload while editing a single book, run `mdbook serve` inside `docs/user-guide/` or `docs/dev-guide/`. Note that running `mdbook` directly uses the default mdBook theme without the shared `docs/theme/` language switcher.

To preview the Chinese edition (the default in `book.toml`):

```bash
mdbook serve --open
```

To preview the English edition in Bash:

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_BOOK__TITLE="cjv User Guide" mdbook serve --open
```

To preview the English edition in PowerShell:

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
$env:MDBOOK_BOOK__TITLE = "cjv User Guide"
mdbook serve --open
```

Use `"cjv Developer Guide"` when previewing `docs/dev-guide/`. Clear those three environment variables when you finish so future `mdbook` commands return to the Chinese defaults.

## Pages deployment layout

`.github/workflows/pages.yml` runs when documentation, frontend, or deployment files change on `master`, on manual workflow dispatch, or when called by `.github/workflows/release.yml`. It builds the complete website into `web/dist/` in this order:

```text
web/dist/
  index.html, install.sh, install.ps1, assets/...   # 1. Built by pnpm build in web/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]   # 2. Extracted from release archives
  book/user-guide/{zh-CN,en}/                       # 3. Built by python3 scripts/build-docs.py
  book/dev-guide/{zh-CN,en}/                        # 4. Built by python3 scripts/build-docs.py
```

> **CAUTION:** Always run `pnpm build` in `web/` **before** extracting `dl/` binaries or running `python3 scripts/build-docs.py`, because `pnpm build` empties `web/dist/`.

The workflow packages `web/dist/` as a single `github-pages` artifact and deploys it to both GitHub Pages (`cjv.zxilly.dev`) and Tencent Cloud EdgeOne (`cjv.zxilly.com`). Read [Landing page](web.md) and [Releases](releasing.md) for how the frontend and release binaries integrate with the documentation site.
