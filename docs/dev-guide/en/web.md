# Landing page

`web/` contains the static landing page at [cjv.zxilly.com](https://cjv.zxilly.com) and [cjv.zxilly.dev](https://cjv.zxilly.dev), with installer commands and platform-specific downloads. It uses React Router Framework Mode, Vite, TypeScript, Tailwind CSS, and Lingui. `web/package.json` declares dependency ranges and the pnpm version; `web/pnpm-lock.yaml` locks dependency versions.

## Development and files

To start the local development server, run inside `web/`:

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` generates route types, checks TypeScript, and runs the official React Router build. `react-router.config.ts` sets `ssr: false` and `prerender: ['/', '/en/']`, producing static files in `web/dist/client/`. `pnpm preview` serves that output locally. Development requires Node.js 22.22.0 or newer; production needs no React SSR server.

| File or directory | Contents |
| --- | --- |
| `react-router.config.ts`, `src/routes.ts` | Official prerender configuration and route table |
| `src/root.tsx` | HTML document, SEO metadata, and framework hydration components |
| `src/App.tsx` | Installation landing page layout, mirror toggle, and download interactions |
| `src/hooks/use-platform.ts` | Browser platform detection and download recommendations |
| `src/generated/platforms.ts` | Frontend platform catalog generated from Go (`internal/target/catalog.go`) |
| `src/lib/i18n.ts`, `src/locales/` | Language selection and Lingui message catalogs |
| `src/style.css`, `src/components/` | Theme tokens and UI components |
| `public/install.sh`, `public/install.ps1` | Public Unix and PowerShell installer scripts |

To change the set of supported release platforms, update `internal/target/catalog.go` in Go and run `go generate ./...` from the repository root—never edit `src/generated/platforms.ts` by hand.

## Platform detection

Production builds prerender React into Chinese `/` and English `/en/` HTML through React Router Framework Mode. No React SSR server runs in production. The initial hydration uses the same generic installation methods as the build; platform detection and regional mirror defaults apply afterwards. Content, commands and documentation links remain readable without JavaScript.

`src/lib/seo.ts` defines localized page metadata and `src/components/project-guide.tsx` contains the bilingual getting-started links. Both domains use canonical and hreflang URLs on `.com`, while `.dev` retains its installer commands and downloads. The `mdbook-sitemap` backend generates chapter sitemaps. `scripts/site_seo.py` adds page metadata, normalizes canonical paths and merges the plugin output with the two landing pages into the site sitemap.

`use-platform.ts` computes an initial platform match synchronously from standard browser properties (`navigator.userAgent` and `navigator.platform`), and then queries User-Agent Client Hints (`navigator.userAgentData`) for additional CPU architecture metadata. Detection resolves to one of three states: `ready`, `unknown`, or `unsupported`.

- **macOS architecture ambiguity** — Many macOS browsers report an Intel (`x86_64`) User-Agent even on Apple Silicon (`arm64`) Macs. When a browser cannot reliably confirm the Mac's CPU architecture, the `install.sh` script command remains available (since the shell script detects the native architecture locally), while manual downloads present both Apple Silicon and Intel options side by side.
- **Mobile and tablet browsers** — Mobile operating systems and architectures without prebuilt binaries are marked `unsupported`, including iPadOS browsers running in desktop User-Agent mode.
- **HarmonyOS and OpenHarmony** — HarmonyOS detection prioritizes the version in the `OpenHarmony` / `HarmonyOS` token and preserves its version namespace (`ArkWeb`, `HuaweiBrowser`, and `Chrome` versions are never treated as OS versions). `Phone` / `Mobile` tokens identify phones, and `Tablet` alone identifies ordinary tablets (`unsupported`). `Tablet` combined with `Windows NT` identifies a 2-in-1 device in tablet mode, while `PC` or desktop `Windows NT` tokens identify PCs. HarmonyOS PCs and 2-in-1 devices receive the shell installer command and native `ARM64` / `x86_64` download choices, whereas phones, ordinary tablets, and unclassified devices show device-specific guidance.

Compatibility tokens (`Windows NT`, `Linux`, `Android`) and UA Client Hints are never allowed to override HarmonyOS with Windows or Android, nor are they trusted to guess a HarmonyOS device's native CPU architecture. Instead, the download panel explains how to run `uname -m` in the terminal to choose the right architecture and launch `cjv-init`.

Whenever you modify platform detection, run the real-browser platform tests (`pnpm test:integration` and `pnpm test:built`) and verify that an unknown architecture never produces a single wrong binary recommendation. See [Testing and checks](testing.md#installers-and-frontend).

## Translation and styling

Chinese (`zh`) is the Lingui source language:

- Use `<Trans>` from `@lingui/react/macro` inside JSX.
- Use `useLingui().t` for string expressions inside React components.
- Use `msg` from `@lingui/core/macro` for message descriptors outside components.

After adding or updating UI copy in `web/src/`, extract the message catalogs:

```bash
pnpm i18n:extract
```

Then fill in the new English translations in `src/locales/en/messages.po`. Production page URLs determine the language. Language links update the URL, content and metadata, including browser back/forward navigation. Both development and production use the route language; saved preferences never override an explicit URL. Each render tree owns its Lingui instance so concurrent prerenders cannot mix languages. During production builds, the Lingui Vite plugin compiles the catalogs automatically.

The visual theme is defined in `src/style.css`, dark mode follows the operating system preference, and web fonts are bundled from local dependencies. Animations should continue to respect `useReducedMotion`.

## Deployment contents

During CI deployment, `.github/workflows/pages.yml` builds `web/` first, and then adds platform-specific `cjv-init` binaries under `web/dist/client/dl/` and both bilingual mdBook manuals under `web/dist/client/book/`. A local `pnpm build` in `web/` does not generate those extra directories on its own. See [Releases](releasing.md#website-installers) for `/dl/` installer artifacts and [Documentation site](documentation.md) for the mdBook build.
