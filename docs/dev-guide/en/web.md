# Landing page

`web/` contains the static landing page at [cjv.zxilly.com](https://cjv.zxilly.com) and [cjv.zxilly.dev](https://cjv.zxilly.dev), with installer commands and platform-specific downloads. It uses React, Vite, TypeScript, Tailwind CSS, and Lingui. `web/package.json` declares dependency ranges and the pnpm version; `web/pnpm-lock.yaml` locks dependency versions.

## Development and files

To start the local development server, run inside `web/`:

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` runs TypeScript type-checking before generating the production bundle in `web/dist/`, and `pnpm preview` serves the built output locally.

| File or directory | Contents |
| --- | --- |
| `src/App.tsx` | Installation landing page layout, mirror toggle, and download interactions |
| `src/hooks/use-platform.ts` | Browser platform detection and download recommendations |
| `src/generated/platforms.ts` | Frontend platform catalog generated from Go (`internal/target/catalog.go`) |
| `src/lib/i18n.ts`, `src/locales/` | Language selection and Lingui message catalogs |
| `src/style.css`, `src/components/` | Theme tokens and UI components |
| `public/install.sh`, `public/install.ps1` | Public Unix and PowerShell installer scripts |

To change the set of supported release platforms, update `internal/target/catalog.go` in Go and run `go generate ./...` from the repository root—never edit `src/generated/platforms.ts` by hand.

## Platform detection

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

Then fill in the new English translations in `src/locales/en/messages.po`. At runtime, language selection checks the saved `cjv-lang` preference first, and otherwise picks Chinese or English from the browser's preferred language. During production builds, the Lingui Vite plugin compiles the catalogs automatically.

The visual theme is defined in `src/style.css`, dark mode follows the operating system preference, and web fonts are bundled from local dependencies. Animations should continue to respect `useReducedMotion`.

## Deployment contents

During CI deployment, `.github/workflows/pages.yml` builds `web/` first, and then adds platform-specific `cjv-init` binaries under `web/dist/dl/` and both bilingual mdBook manuals under `web/dist/book/`. A local `pnpm build` in `web/` does not generate those extra directories on its own. See [Releases](releasing.md#website-installers) for `/dl/` installer artifacts and [Documentation site](documentation.md) for the mdBook build.
