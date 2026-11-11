# Documentation site

`docs/user-guide/` and `docs/dev-guide/` are two independent mdBook projects. Each book keeps its Chinese source chapters in `src/` and its English source chapters in `en/`, sharing a single `book.toml` configuration file. Both books share the site theme and language switcher in `docs/theme/`.

## Edit chapters

The Chinese and English chapter sources are maintained side by side—synchronize content changes across both languages in the same commit:

- When adding, moving, or deleting chapters, update both `src/SUMMARY.md` and `en/SUMMARY.md` and keep page file paths identical across both languages.
- Keep CLI commands, file paths, configuration keys, and code identifiers identical; prose and code comments may be translated.
- Use relative `.md` links within the same book, and use full public site URLs (`https://cjv.zxilly.com/book/...`) when linking between the User Guide and the Developer Guide, because the two books build separately.
- Whenever you move or rename a published page, retain its old URL under `[output.seo.html.redirect]` in `book.toml` so existing links continue to work.

`docs/theme/cjv-i18n.js` and `cjv-i18n.css` provide the top-bar language switcher. The switcher works by replacing the `/zh-CN/` and `/en/` path segments in the current URL, which is why both languages must have matching page paths. Because translated headings generate different HTML anchor IDs, switching languages opens the corresponding chapter at the top and strips any previous `#anchor` hash.

## Local preview

### Full site build and link check (recommended)

Install **mdBook 0.5.3**, **mdbook-sitemap 0.1.1**, **mdbook-seo** and **Python 3.9 or later**, then run from the repository root:

```bash
cargo install mdbook-sitemap --version 0.1.1 --locked
cargo install --git https://github.com/Zxilly/mdbook-seo --rev de97785a384305e149dc52ac24f9f1d8f827bb9c --locked
python3 scripts/build-docs.py
python3 -m http.server --directory web/dist/client 8000
```

Open `http://localhost:8000/book/user-guide/en/` (or `/zh-CN/`) in your browser to preview the published theme and language switcher. On Windows, use `python` if `python3` is not on your `PATH`.

`scripts/build-docs.py` configures the source directory, language code, book title, and site URL for all four books (`user-guide` and `dev-guide` in `zh-CN` and `en`), stages the shared theme in a temporary directory so mdBook can fingerprint assets cleanly, and then parses all generated HTML files to verify internal page links, cross-book `/book/` links, static assets, and `#anchor` fragments. Pass `--output <dir>` to write to another output directory, or `--check-only` to check an existing build without running `mdbook`. Markdown lint checks are listed in [Testing and checks](testing.md#documentation-workflows-and-dependencies).

Each book uses `[output.seo]` to render HTML with [mdbook-seo](https://github.com/Zxilly/mdbook-seo), including canonical, description, hreflang, Open Graph and Twitter metadata. HTML renderer options belong under `[output.seo.html]`. The shared `docs/theme/head.hbs` provides analytics, and `docs/user-guide/seo-pages.json` supplies localized installation-page overrides. The build sets each language’s canonical base URL and locale.

The `mdbook-sitemap` backend discovers chapters through `[output.sitemap]`. The full build copies `seo/` output to the existing public paths. `scripts/site_map.py` reads the generated canonical tags, excludes noindex pages, and merges the four books with whichever home and Blog pages have actually been built. It does not modify HTML. Backend directory names do not appear in public URLs. `include-lastmod` remains disabled so build dates are not reported as modification dates.

### Rebuilding during editing

After changing Markdown or SEO configuration, rerun `python3 scripts/build-docs.py` and refresh the static preview. This is the supported preview path for the complete bilingual site; it uses the same staging, theme and metadata as production. A direct `mdbook serve` is not part of the SEO backend’s tested compatibility.

## Pages deployment

`.github/workflows/pages.yml` deploys the site when relevant frontend, documentation, or deployment files are pushed to `master`, on manual `workflow_dispatch` runs, or when called by the release workflow. It builds `web/` first, and then populates installers and all four book outputs under `web/dist/client/`:

```text
web/dist/client/
  dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]
  book/user-guide/{zh-CN,en}/
  book/dev-guide/{zh-CN,en}/
```

Always keep this build order: running `pnpm build` in `web/` clears `web/dist/client/`. Pull request checks and production deployments use the exact same mdBook version and build script entry point:

```bash
python3 scripts/build-docs.py
```

The assembled `web/dist/client` directory is uploaded as a single `github-pages` artifact and deployed to both GitHub Pages and Tencent Cloud EdgeOne. See [Releases](releasing.md) for installer artifacts and release ordering.
