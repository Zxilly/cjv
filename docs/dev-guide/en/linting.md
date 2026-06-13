# Linting and Formatting

cjv runs checks across three languages: Go code, Markdown docs, and the landing page's TypeScript. Each category is gated by a CI job (see [ci.md](ci.md)). This chapter explains what these checks do and how to run them locally beforehand, so you don't push only to discover a red X.

## Go formatting

Go code is formatted uniformly with `gofmt`. Format the whole tree before committing:

```bash
gofmt -w .
```

To see which files aren't formatted without changing them, use `-l` to list only the file names:

```bash
gofmt -l .
```

No output means everything is already formatted. CI has no dedicated gofmt diff check job; formatting relies on developer discipline and the editor's format-on-save. Most Go editor integrations run `gofmt` (or the equivalent `goimports`) on save by default, so with that enabled you rarely need to manage it by hand.

## go vet

`go vet` is Go's built-in static check. It catches things the compiler doesn't flag but that are likely bugs, such as `Printf` format strings that don't match their arguments, or copying a struct that contains a lock. Run it locally:

```bash
go vet ./...
```

cjv has a `mirror` build tag (a variant for domestic mirror sources), and that code is only compiled when the tag is set, so it needs a separate vet pass:

```bash
go vet -tags=mirror ./...
```

These two correspond to the `Run go vet` and `Run go vet (mirror)` steps of the `go-test` job in CI.

## golangci-lint

More thorough Go checks are handled by `golangci-lint`. The config is in [`.golangci.yml`](https://github.com/Zxilly/cjv/blob/master/.golangci.yml), using the v2 config format. For local installation see the [official docs](https://golangci-lint.run/welcome/install/); once installed, run it from the repository root:

```bash
golangci-lint run
```

In the config, `default: standard` enables golangci-lint's standard set of linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, and `unused`. On top of that, three more are enabled:

- `errorlint`: checks the use of Go 1.13 error wrapping, for example using `==` where `errors.Is` should be used.
- `copyloopvar`: checks for redundant copying of loop variables (such code can be simplified after Go 1.22 changed loop variable semantics).
- `errname`: requires sentinel errors to be named with an `Err` prefix and error types to be named with an `Error` suffix.

Two exceptions are written into the config. `errcheck` ignores the `progressbar` progress bar's `Add64` and `Finish`, whose returned errors are purely cosmetic and pointless to handle. Test files (`_test.go`) relax `errcheck` entirely, since unchecked errors from things like `defer f.Close()` are the norm in tests.

`golangci-lint` can fix some problems automatically:

```bash
golangci-lint run --fix
```

CI's `lint` job runs via `golangci-lint-action`, with the version pinned to `latest`. If the version you have installed locally is older, it may report a few fewer or more findings; the CI result is authoritative.

## Markdown

The Markdown in the repository (the two READMEs and the mdBook manuals) is checked with `markdownlint-cli2`. The config is in [`.markdownlint-cli2.jsonc`](https://github.com/Zxilly/cjv/blob/master/.markdownlint-cli2.jsonc) at the repository root. Run it from the repository root:

```bash
npx markdownlint-cli2
```

No prior installation is needed; `npx` fetches a copy on the fly. The scope is defined by `globs` in the config: `README.md`, `README.EN.md`, and `docs/**/*.md`. The two `SUMMARY.md` files are excluded via `ignores`, because mdBook uses multiple top-level headings as group titles, which trips MD025 (only one top-level heading allowed per document); that's a structural requirement, not a writing problem.

A few rules are turned off or adjusted in the config, all in service of Chinese-language docs:

- `MD013` (line length) is turned off. Chinese paragraphs don't wrap, so a line-length limit makes no sense for CJK body text, tables, or long URLs.
- `MD060` (table column alignment) is turned off. This rule wants pipes aligned by character count, but Chinese characters are full-width and take two character widths, so aligning by character count actually looks misaligned.
- `MD024` (duplicate headings) is set to `siblings_only`, allowing sections with the same name under different parent headings, for example multiple chapters each having an "Examples" section.

CI's `markdown-lint` job runs `npx --yes markdownlint-cli2@0.22.1`, with the version pinned. Running `npx markdownlint-cli2` locally gives you the latest version, whose rule set occasionally differs; to match CI exactly, pass the version explicitly:

```bash
npx --yes markdownlint-cli2@0.22.1
```

## Landing page TypeScript

Type checking for the landing page (`web/`, see [web.md](web.md)) is done by the TypeScript compiler `tsc`, which is wired into the build. The `build` script in `web/package.json` is `tsc -b && vite build`, so a single build type-checks first and then bundles, and if the types don't pass it fails outright. Run it from the `web/` directory:

```bash
cd web
pnpm install
pnpm build
```

CI's `web-build` job does exactly this: `pnpm install --frozen-lockfile` followed by `pnpm build`. The landing page has no separate ESLint job; type safety relies on `tsc`, and behavioral correctness relies on tests (see [testing.md](testing.md)).

## Dependency checks

Two checks related to Go modules are also in CI and can be reproduced locally.

`go mod tidy` ensures `go.mod` and `go.sum` are clean, with no extra or missing dependencies. CI's `mod-check` job runs `go mod tidy` and then uses `git diff --exit-code` to verify the files weren't changed; if they were, what was committed wasn't in a tidy state. Run the same thing locally:

```bash
go mod tidy
git diff --exit-code go.mod go.sum
```

The same job also runs `go mod verify`, which checks that the dependencies in the local module cache match the checksums from when they were downloaded and haven't been tampered with:

```bash
go mod verify
```

## Vulnerability scanning

`govulncheck` scans for known vulnerabilities in the dependencies your code actually calls. It only reports call paths that are genuinely reachable, so the noise is very low. CI's `vuln-check` job does a `go install` and then scans. Install it once locally:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Then scan the whole tree:

```bash
govulncheck ./...
```

## One-shot local self-check

To run all of the above at once before pushing, execute the following in order from the repository root:

```bash
gofmt -l .
go vet ./...
go vet -tags=mirror ./...
golangci-lint run
go mod tidy && git diff --exit-code go.mod go.sum
go mod verify
govulncheck ./...
npx --yes markdownlint-cli2@0.22.1
```

For changes that touch the landing page, add this step:

```bash
cd web && pnpm install && pnpm build
```

This sequence of commands covers the check portions of the `go-test`, `lint`, `mod-check`, `vuln-check`, `markdown-lint`, and `web-build` jobs in CI. Getting it all green basically guarantees these jobs won't fail on checks or formatting, leaving the rest to tests (see [testing.md](testing.md)).
