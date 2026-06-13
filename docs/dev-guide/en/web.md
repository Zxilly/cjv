# Landing Page

The landing page is the root page of [cjv.zxilly.dev](https://cjv.zxilly.dev), with its source in the repository's `web/` directory. It is a static single-page application whose core job is to detect the visitor's operating system and CPU architecture and then offer the most suitable install command or download link. The page has no backend; all the logic runs in the browser, and the build output is a set of static files hosted by GitHub Pages.

The documentation site and the landing page are deployed together on the same Pages site: the landing page occupies the site root, while the user guide and this development guide are built into `web/dist/book/` as sub-paths. This section only covers the landing page itself. For building the documentation site see [documentation.md](documentation.md), and for the full Pages deployment flow see [ci.md](ci.md).

## Tech stack

The landing page is bundled with Vite, the UI is written with React 19, styling uses Tailwind CSS v4, and animation uses framer-motion. The package manager is pnpm, pinned in the `packageManager` field of `web/package.json` (`pnpm@10`), and CI uses Node 22.

`web/vite.config.ts` chains three plugins together:

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { lingui } from '@lingui/vite-plugin'
import path from 'node:path'

export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  plugins: [
    react({ babel: { plugins: ['@lingui/babel-plugin-lingui-macro'] } }),
    tailwindcss(),
    lingui(),
  ],
})
```

`@lingui/babel-plugin-lingui-macro` is hooked into `@vitejs/plugin-react`, so that Lingui's macros (covered below) can be expanded at compile time. `@` is a path alias for `src/`, so imports are written like `@/hooks/use-platform`; the same alias is also configured in `web/vitest.config.ts` so it can be resolved when running tests.

Tailwind v4 does not need a `tailwind.config.js`; the theme is written directly in CSS. The entry stylesheet is `web/src/style.css`: `@import "tailwindcss"` at the top pulls in the framework, and the `@theme` block defines custom tokens such as the brand color `--color-cj` (which maps to the utility classes `text-cj` and `bg-cj`) and the font stack. Dark mode uses `@custom-variant dark` together with the `prefers-color-scheme` media query; there is no manual toggle. Fonts are bundled into the output via `@import` from the `@fontsource` packages, with no dependency on an external CDN.

The UI structure is concentrated in `web/src/App.tsx`, the main component for the whole page: the title, the install card (with three tabs for command install / download install / build-from-source install), the mirror source toggle, and the footer navigation all live here. Reusable small components are in `web/src/components/`, and under `components/ui/` are unstyled primitives wrapping `radix-ui` (`tabs`, `switch`, `collapsible`, and so on). framer-motion handles animations such as the sliding tab switch and the card-height transition, and it uniformly respects the system "reduce motion" preference through `useReducedMotion`; when that preference is on, all transitions degrade to zero duration.

## Internationalization

The landing page uses [Lingui](https://lingui.dev) for i18n. The source language is Simplified Chinese and English is the translation, which is consistent with the rest of the project: you write Chinese directly in the code, and the English translation is provided by the po catalog. The configuration is in `web/lingui.config.ts`:

```ts
import { defineConfig } from '@lingui/cli'

export default defineConfig({
  sourceLocale: 'zh',
  locales: ['zh', 'en'],
  catalogs: [
    {
      path: '<rootDir>/src/locales/{locale}/messages',
      include: ['<rootDir>/src'],
    },
  ],
})
```

The catalogs land in `web/src/locales/zh/messages.po` and `web/src/locales/en/messages.po`. Because the source language is Chinese, each entry's `msgstr` in the `zh` catalog is identical to its `msgid`; only in the `en` catalog is the `msgstr` the actual English translation.

### Writing translatable strings in code

Lingui collects strings at compile time through macros, so you write macros rather than maintaining IDs by hand. The landing page uses three of them, chosen by context:

In JSX, use `<Trans>`, imported from `@lingui/react/macro`. It can wrap text that contains child elements, and the placeholder tags get numbered:

```tsx
import { Trans } from '@lingui/react/macro'

<Trans>使用 Go 从源码编译安装：</Trans>
```

When you need the string at runtime (for example to pass it to a `title` attribute or for a conditional check), use the `t` template tag returned by `useLingui()`, also from `@lingui/react/macro`:

```tsx
import { useLingui } from '@lingui/react/macro'

const { t } = useLingui()
const title = t`其他平台`
```

If a string needs to be defined outside a component, at module top level (there are quite a few of these in `web/src/hooks/use-platform.ts`), use `msg` from `@lingui/core/macro`. It produces a `MessageDescriptor`, whose evaluation is deferred until render time via `i18n._(descriptor)`:

```ts
import { msg } from '@lingui/core/macro'

const MAC_X86_WARNING = msg`部分 LTS 和 STS 版本可能不包含 macOS x86_64 的预编译 SDK。`
```

### After adding a string

After writing new Chinese text, run the extract command to sync it into the catalogs:

```bash
cd web
pnpm i18n:extract
```

This scans all macros under `src/` and writes new `msgid`s into both `messages.po` files. For a new entry, the `msgstr` in the `zh` catalog automatically equals the Chinese source, while in the `en` catalog the `msgstr` is empty and needs to be filled in with English by you (or by a later translation workflow). Please complete the `en` translations before committing, and do not leave empty `msgstr`s. `pnpm i18n:compile` is a separate script; during a production build the Lingui Vite plugin processes the `.po` files directly, so you generally do not need it in day-to-day development.

Runtime language loading lives in `web/src/lib/i18n.ts`: when the module loads it calls `detectLang()` to pick a language (preferring `cjv-lang` stored in `localStorage`, otherwise checking whether `navigator.language` is `zh`, with English as the default fallback), then `activateLang` activates the corresponding catalog. The top level of `web/src/App.tsx` uses `<I18nProvider>` to inject the `i18n` instance into the React tree, and the language toggle button in the top-right corner calls `activateLang` + `persistLang` to switch in real time and remember the choice.

## Platform detection

Platform detection is the core of the landing page. The logic is all in `web/src/hooks/use-platform.ts`, pure functions plus a `usePlatform` hook. The question it answers is: what system and architecture is the visitor on, and which install command should we give them.

Detection happens in two steps. On the first render it synchronously computes a preliminary result (`computeBrowserPlatformResult`) using immediately available information such as `navigator.platform` and `navigator.userAgent`, parsed with `ua-parser-modern`. After mounting it makes another asynchronous call (`detectBrowserPlatformResult`): if the browser supports UA Client Hints (`navigator.userAgentData.getHighEntropyValues`), it requests the more precise `architecture`, `bitness`, and `platform`, then recomputes once it has them. Between the two steps, `samePlatformResult` compares the user-visible fields, and if they are the same it skips the re-render.

The detection result is a discriminated union `PlatformResult`, with three possible values for `state`:

- `ready`: a supported platform was identified. In most cases it also carries a specific `binary` (a directly downloadable link). There is one exception: Safari and Firefox on macOS do not expose the CPU architecture, in which case the state is still `ready` (because `install.sh` resolves the architecture itself), but `binary` is `null`, and the UI instead asks the user to choose between Apple Silicon and Intel.
- `unsupported`: a known mobile system (iOS / Android / HarmonyOS), or a known desktop system whose architecture has no prebuilt version (for example Windows arm64). `info.reason` will be `'mobile'` or `'arch'`, and the UI gives different advice accordingly.
- `unknown`: not recognized, so it simply lists the install methods for all platforms.

Which platforms are supported is not hardcoded in the frontend. `web/src/generated/platforms.ts` is a generated file, exported from the Go-side `internal/target` package by `scripts/gen-platform-surfaces.go`, with a `DO NOT EDIT` marker at the top. This keeps the platform list shown by the frontend always consistent with the build targets cjv actually supports; after changing the support matrix on the Go side, running `go generate ./...` (the hook is in `internal/target/catalog.go`) syncs it over, so there is no need to maintain two copies by hand. `PLATFORM_PRESENTATION` in `use-platform.ts` is only responsible for adding presentation-related labels, hint text, and install commands to these platforms.

HarmonyOS recognition has a dedicated piece of handling: `ua-parser-modern` only reports `HarmonyOS` when the UA carries both the `android` and `harmonyos` markers, but HarmonyOS NEXT drops the `android` marker and gets misidentified, so the code additionally matches `harmonyos`/`openharmony` in the raw UA as a fallback. iPadOS in "desktop mode" disguises itself as macOS, and the code uses `platform === 'MacIntel'` plus `maxTouchPoints > 1` to recognize it back as iOS. These are real edge cases that are documented in comments, so be careful not to break them when changing this area.

For the background on why all of this guessing is done in the browser instead of reading the architecture directly on the server, see the note about macOS architecture detection in auto-memory. The core conclusion is that the browser cannot reliably read a Mac's CPU architecture, which is why there is the fallback design of `install.sh` plus a two-choice download.

## Local development

All commands are run in the `web/` directory. On your first time here, install dependencies first:

```bash
cd web
pnpm install
```

Common scripts (defined in `web/package.json`):

```bash
pnpm dev      # start the Vite dev server with hot reload
pnpm build    # tsc -b type check + vite build, output in web/dist
pnpm preview  # preview the built output in web/dist locally
```

`pnpm build` first runs `tsc -b` for a full type check before bundling, so a type error will fail the build outright.

Tests use Vitest and run in a real browser (`@vitest/browser` + Playwright), not jsdom. The configuration is in `web/vitest.config.ts`, the default browser is chromium, and you can switch to `firefox` or `webkit` with the `VITEST_BROWSER` environment variable. Common scripts:

```bash
pnpm test       # run all tests once
pnpm test:watch # watch mode
pnpm coverage   # run tests and produce a coverage report (80% threshold)
```

Tests run in the browser, so before the first run you need to install Playwright's browsers: `pnpm exec playwright install --with-deps chromium`. `pnpm test:integration` separately runs `src/App.platform.integration.test.tsx`, which is an integration test that verifies end-to-end platform detection using real browser UAs; CI runs it under a matrix of the three engines chromium, firefox, and webkit across multiple runner operating systems. For the overall testing conventions see [testing.md](testing.md).

On the CI side, `.github/workflows/ci.yml` has the jobs `web-build`, `web-test`, `web-coverage`, and `web-browser-integration`, corresponding to the commands above. Deployment is handled by `.github/workflows/pages.yml`: it first runs `pnpm build` to produce `web/dist`, then unpacks the per-platform `cjv-init` binaries into `web/dist/dl/` for the page's download links to use, then builds the user guide and development guide into `web/dist/book/`, and finally publishes the whole `web/dist` as a Pages artifact. See [ci.md](ci.md) for details.
