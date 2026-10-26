# Landing page

The `web/` directory contains the static landing page deployed at [cjv.zxilly.com](https://cjv.zxilly.com) and [cjv.zxilly.dev](https://cjv.zxilly.dev). It displays platform-specific installer commands and direct binary downloads. The project uses React, Vite, TypeScript, Tailwind CSS, and Lingui, with exact dependency and `pnpm` versions pinned in `web/package.json`.

## Local development and file layout

Run these commands inside `web/` to start the local development server:

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

- `pnpm build` runs the TypeScript compiler check and builds the static bundle into `web/dist/`.
- `pnpm preview` serves the built `web/dist/` directory locally.

| File or directory | Purpose |
| --- | --- |
| `src/App.tsx` | Main landing page layout, mirror toggle, and download interactions |
| `src/hooks/use-platform.ts` | Browser platform and CPU architecture detection logic |
| `src/generated/platforms.ts` | Platform catalog generated from Go (`internal/target/catalog.go`) |
| `src/lib/i18n.ts`, `src/locales/` | Language detection, switcher state, and Lingui `.po` catalogs |
| `src/style.css`, `src/components/` | Tailwind CSS theme tokens and UI components |
| `public/install.sh`, `public/install.ps1` | Public Unix and PowerShell installer scripts |

> **CAUTION:** Never edit `web/src/generated/platforms.ts` by hand. To add or change a supported platform, update `internal/target/catalog.go` and run `go generate ./...` from the repository root; read [Build from source](building.md#cross-compilation-and-generated-platform-catalogs).

## Browser platform detection

### Detection states and macOS architecture handling

`use-platform.ts` computes an initial platform match synchronously from `navigator.userAgent` and `navigator.platform`, and then queries User-Agent Client Hints (`navigator.userAgentData`) for high-entropy CPU architecture data. It classifies the visitor's environment into one of three states:

- `ready`: cjv supports both the operating system and the CPU architecture.
- `unknown`: cjv supports the operating system, but the browser cannot confirm the CPU architecture.
- `unsupported`: mobile operating systems or CPU architectures without prebuilt `cjv` binaries.

On macOS, many browsers report an Intel (`x86_64`) user agent even on Apple Silicon (`arm64`) hardware. When browser hints do not confirm the exact Mac architecture, the page still shows the `install.sh` one-liner (which detects the architecture locally in the shell) and presents both Apple Silicon and Intel buttons for manual archive downloads. Detection also handles iPadOS desktop-class user agents.

### HarmonyOS and OpenHarmony detection rules

For HarmonyOS and OpenHarmony visitors, the detector applies strict token rules:

1. **OS version priority**: reads the version number from the `OpenHarmony` or `HarmonyOS` token and keeps its version namespace intact. It never mistakes `ArkWeb`, `HuaweiBrowser`, or `Chrome` version numbers for the OS version.
2. **Form-factor classification**:
   - `Phone` or `Mobile` identifies a phone (`unsupported`).
   - `Tablet` together with `Windows NT` identifies a 2-in-1 device in tablet mode (`ready` or `unknown`).
   - `PC` or a desktop `Windows NT` token identifies a HarmonyOS PC (`ready` or `unknown`).
   - `Tablet` alone identifies an ordinary tablet (`unsupported`).
3. **Architecture safety**: compatibility tokens (`Windows NT`, `Linux`, `Android`) and UA Client Hints do not override HarmonyOS with Windows or Android, and do not establish the native CPU architecture of a HarmonyOS PC. Instead, the download panel instructs the user to run `uname -m` in HiShell to choose `ARM64` or `x86_64` and run `cjv-init`.

Whenever you change `use-platform.ts`, run the browser integration and built-page test suites (`pnpm test:integration` and `pnpm test:built`) to make sure that an unknown architecture never recommends a single wrong binary. Read [Tests and checks](testing.md#installer-scripts-and-web-frontend).

## Translation and styling

Chinese (`zh`) is the Lingui source language:

- Use `<Trans>` from `@lingui/react/macro` for JSX elements.
- Use `useLingui().t` for string expressions inside React components.
- Use `msg` from `@lingui/core/macro` for message descriptors outside components.

After adding or changing any UI text in `web/src/`, extract the message catalog:

```bash
pnpm i18n:extract
```

Then fill in the English translations in `web/src/locales/en/messages.po`. At runtime, the page checks the saved `cjv-lang` preference first and falls back to the browser's preferred language (`zh` or `en`). During `pnpm build`, the Lingui Vite plugin compiles the catalogs automatically.

Theme colors and typography live in `src/style.css`, dark mode follows the OS color scheme, and the build bundles web fonts from local `node_modules` packages. All animations must respect `useReducedMotion`.

## Production bundle and Pages deployment

Running `pnpm build` in `web/` produces the landing page and `install.sh` / `install.ps1` only. During CI deployment, `.github/workflows/pages.yml` adds two more sets of files into `web/dist/`:

- `/dl/{official,mirror}/{os}_{arch}/cjv-init[.exe]` standalone installer binaries extracted from the latest release; read [Releases](releasing.md#website-installers).
- `/book/user-guide/{zh-CN,en}/` and `/book/dev-guide/{zh-CN,en}/` built by `scripts/build-docs.py`; read [Documentation site](documentation.md).
