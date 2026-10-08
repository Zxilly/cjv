# Landing page

`web/` is the static site at [cjv.zxilly.dev](https://cjv.zxilly.dev), providing installer commands and platform downloads. It uses React, Vite, TypeScript, Tailwind CSS, and Lingui. `web/package.json` defines dependencies and the pnpm version.

## Development and files

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

`pnpm build` runs TypeScript checks before generating `dist/`. `pnpm preview` serves the built output.

| File or directory | Contents |
| --- | --- |
| `src/App.tsx` | Installation page and interactions |
| `src/hooks/use-platform.ts` | Browser platform detection and download suggestions |
| `src/generated/platforms.ts` | Frontend data generated from the Go platform catalog |
| `src/lib/i18n.ts`, `src/locales/` | Language selection and translations |
| `src/style.css`, `src/components/` | Theme and components |
| `public/install.sh`, `public/install.ps1` | Installer scripts |

To change supported platforms, update `internal/target` in Go and run `go generate ./...`. Do not edit the generated TypeScript directly.

## Platform detection

The page computes a result from synchronous browser information, then tries UA Client Hints for additional architecture data. Results are `ready`, `unsupported`, or `unknown`.

When a macOS browser cannot reliably report CPU architecture, the installer script remains available while manual downloads offer Apple Silicon and Intel choices. Mobile systems and architectures without prebuilt binaries are marked unsupported. Detection also handles iPadOS desktop mode and HarmonyOS NEXT user agents.

HarmonyOS detection prioritizes the version in the `OpenHarmony` / `HarmonyOS` token and preserves its version namespace. ArkWeb, HuaweiBrowser, and Chrome versions are not OS versions. `Phone` / `Mobile` identifies phones and `Tablet` identifies tablets; based on collected UA samples, `Tablet` together with `Windows NT` selects guidance for a 2-in-1 in tablet mode. Compatibility tokens and UA Client Hints cannot override HarmonyOS with Windows or Android, and do not establish its native CPU architecture. HarmonyOS remains unsupported with no recommended installer; a TODO marks the future installation support entry point.

After changes here, run real-browser platform tests and verify that an unknown architecture does not produce a single incorrect binary recommendation. See [testing and checks](testing.md).

## Translation and styling

Chinese is the Lingui source language. Use `<Trans>` from `@lingui/react/macro` in JSX, `useLingui().t` for strings in components, and `msg` from `@lingui/core/macro` for descriptors outside components. After adding copy:

```bash
pnpm i18n:extract
```

Fill new English translations in `src/locales/en/messages.po`. Runtime selection uses saved `cjv-lang` first, then chooses Chinese or English from the browser language. The Lingui Vite plugin processes catalogs during production builds.

The theme is defined in CSS, dark mode follows the system, and fonts are bundled from local dependencies. Motion should continue to respect `useReducedMotion`.

## Deployment contents

Pages adds platform-specific `cjv-init` binaries and both bilingual books after the frontend build. A local Vite build does not generate these files. See [releases](releasing.md) for download artifacts and [documentation](documentation.md) for book outputs.
