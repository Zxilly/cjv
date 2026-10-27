# Documentation site

`docs/user-guide/` and `docs/dev-guide/` are two independent mdBook projects. Each book keeps its Chinese source chapters in `src/` and its English source chapters in `en/`, sharing a single `book.toml` configuration file. Both books share the site theme and language switcher in `docs/theme/`.

## Edit chapters

The Chinese and English chapter sources are maintained side by side—synchronize content changes across both languages in the same commit:

- When adding, moving, or deleting chapters, update both `src/SUMMARY.md` and `en/SUMMARY.md` and keep page file paths identical across both languages.
- Keep CLI commands, file paths, configuration keys, and code identifiers identical; prose and code comments may be translated.
- Use relative `.md` links within the same book, and use full public site URLs (`https://cjv.zxilly.com/book/...`) when linking between the User Guide and the Developer Guide, because the two books build separately.
- Whenever you move or rename a published page, retain its old URL under `[output.html.redirect]` in `book.toml` so existing links continue to work.

`docs/theme/cjv-i18n.js` and `cjv-i18n.css` provide the top-bar language switcher. The switcher works by replacing the `/zh-CN/` and `/en/` path segments in the current URL, which is why both languages must have matching page paths. Because translated headings generate different HTML anchor IDs, switching languages opens the corresponding chapter at the top and strips any previous `#anchor` hash.

## Writing conventions

- Use factual, direct descriptions of command behavior, prerequisites, defaults, and limitations.
- Keep the project name, purpose, installation, and quick start in the README. Put detailed options in the user guide or command reference.
- Use short topic or task headings. Avoid promotional claims, repeated chapter introductions, and decorative emphasis on links.
- Base behavior descriptions on code and tests. Distinguish defaults, explicit options, and platform differences, and keep both languages consistent.

## Local preview

### Full site build and link check (recommended)

Install **mdBook 0.5.3** and **Python 3.9 or later**, then run from the repository root:

```bash
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist 8000
```

Open `http://localhost:8000/book/user-guide/en/` (or `/zh-CN/`) in your browser to preview the published theme and language switcher. On Windows, use `python` if `python3` is not on your `PATH`.

`scripts/build-docs.py` configures the source directory, language code, book title, and site URL for all four books (`user-guide` and `dev-guide` in `zh-CN` and `en`), stages the shared theme in a temporary directory so mdBook can fingerprint assets cleanly, and then parses all generated HTML files to verify internal page links, cross-book `/book/` links, static assets, and `#anchor` fragments. Pass `--output <dir>` to write to another output directory, or `--check-only` to check an existing build without running `mdbook`. Markdown lint checks are listed in [Testing and checks](testing.md#documentation-workflows-and-dependencies).

### Previewing a single book with `mdbook serve`

If you want live reload while editing a single book's prose, you can run `mdbook serve` directly inside `docs/user-guide/` or `docs/dev-guide/` (note that this entry point uses the default mdBook theme without the shared language switcher):

```bash
mdbook serve --open
```

To preview the English edition in Bash:

```bash
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en MDBOOK_BOOK__TITLE="cjv User Guide" mdbook serve --open
```

In Windows PowerShell:

```powershell
$env:MDBOOK_BOOK__SRC = "en"
$env:MDBOOK_BOOK__LANGUAGE = "en"
$env:MDBOOK_BOOK__TITLE = "cjv User Guide"
mdbook serve --open
```

Use `"cjv Developer Guide"` when previewing the developer guide. Remember to unset those three environment variables afterward to restore the Chinese defaults, and replace `serve --open` with `build` if you want to build without starting a local server.

## Pages deployment

`.github/workflows/pages.yml` deploys the site when relevant frontend, documentation, or deployment files are pushed to `master`, on manual `workflow_dispatch` runs, or when called by the release workflow. It builds `web/` first, and then populates installers and all four book outputs under `web/dist/`:

```text
web/dist/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

Always keep this build order: running `pnpm build` in `web/` clears `web/dist/`. Pull request checks and production deployments use the exact same mdBook version and build script entry point:

```bash
python3 scripts/build-docs.py
```

The assembled `web/dist` directory is uploaded as a single `github-pages` artifact and deployed to both GitHub Pages and Tencent Cloud EdgeOne. See [Releases](releasing.md) for installer artifacts and release ordering.
