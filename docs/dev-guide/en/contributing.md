# Contributing Guide

Welcome to contributing to cjv. Whether you are fixing a typo, filling out a piece of documentation, or implementing a new subcommand, the flow is the same: fork the repository, open a branch, run the checks locally, and open a PR. This chapter spells out that flow so your first contribution merges smoothly.

## Fork and branch

cjv is hosted at [github.com/Zxilly/cjv](https://github.com/Zxilly/cjv), with `master` as the main branch. Fork it to your own account on GitHub, then clone it:

```bash
git clone https://github.com/<your-username>/cjv.git
cd cjv
git remote add upstream https://github.com/Zxilly/cjv.git
```

Do not edit `master` directly. Open a separate branch for each change, cut from the latest `upstream/master`:

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

There is no enforced naming convention for branches; just give it a short name that conveys the intent. Keep one PR to one thing, so review is fast and rollback is clean.

See [building.md](building.md) for the build and runtime environment, and [architecture.md](architecture.md) for the overall structure of the code.

## Run the checks before you change anything

Before you commit, run the checks that CI runs locally, so you don't have to wait on the pipeline back and forth. The full CI definition lives in `.github/workflows/ci.yml`; the ones you'll use most often locally are below.

For Go code, build it first, run the tests, and run vet:

```bash
go build ./...
go test -count=1 ./...
go vet ./...
```

cjv has a `mirror` build tag (for mirror sources in mainland China). If your change touches the download or self-update logic, build the tagged variant as well while you're at it:

```bash
go build -tags=mirror ./...
```

Code style is enforced with `golangci-lint`, configured in `.golangci.yml`:

```bash
golangci-lint run
```

For how tests are organized and how to run the integration tests, see [testing.md](testing.md); for the details of linting and formatting, see [linting.md](linting.md).

If you changed `go.mod` or your dependencies, run `go mod tidy` and confirm there are no leftover changes; CI's `mod-check` job enforces this with `git diff --exit-code go.mod go.sum`:

```bash
go mod tidy
git diff --exit-code go.mod go.sum
```

If you changed any Markdown (including the docs and the README), run it through markdownlint; the rules are in `.markdownlint-cli2.jsonc`:

```bash
npx --yes markdownlint-cli2
```

Changes to the landing page (the `web/` directory) have their own build and tests; see [web.md](web.md).

## Commit message conventions

The repository follows the [Conventional Commits](https://www.conventionalcommits.org/) style, with the format `type(scope): description`. The `scope` is optional. Look at a few past commits to get a feel for it:

```text
feat(toolchain): support installing from a URL via `toolchain link`
fix(env): derive Windows env-script bin dir from script location
docs: add a bilingual user manual and slim the README
refactor(install): extract lifecycle orchestration
test(windows): ignore registry key close errors
ci: lint Markdown with markdownlint-cli2
chore(deps): bump golang.org/x/crypto to v0.52.0
```

The `type` values actually used in this repository:

- `feat`: a new feature
- `fix`: a bug fix
- `docs`: documentation-only changes
- `refactor`: a refactor that does not change external behavior
- `test`: adding or changing tests
- `ci`: changes to the CI configuration
- `chore`: miscellaneous work (dependency bumps, cleanup, and so on)

The `scope` marks which subsystem the change lands in; common ones are `cli`, `web`, `install`, `toolchain`, `env`, `config`, `ci`, and so on. When a change spans two scopes, it can also be written as `fix(proxy,cli): ...`. The `scope` is not a fixed enumeration; just match it to the change, and omit it if you're unsure.

Write the description as an imperative phrase, lowercase, with no trailing period, summarizing what the commit does. Commit messages are not enforced by a commit hook, but please follow the style above to keep the history readable.

## Opening a PR

Push your branch to your fork, then open a PR targeting the `master` branch of `Zxilly/cjv`:

```bash
git push -u origin fix-windows-path
```

You can also open it directly with the `gh` command line:

```bash
gh pr create --repo Zxilly/cjv --base master
```

In the PR description, explain clearly what you changed and why; if it corresponds to an issue, include a reference. Once the PR is open, CI automatically runs the full set of checks: Go is built and tested on Linux, macOS, and Windows across amd64 and arm64 (including `-race`), plus `golangci-lint`, markdownlint, the `go mod tidy` check, the `govulncheck` vulnerability scan, cross-platform cross-compilation, and the landing page's build and browser integration tests. The install script also has a set of integration tests covering several shells. For the details of these jobs, see [ci.md](ci.md).

Every check has to be green before a PR is merged. If a job goes red, click into it to read the log, reproduce it locally and fix it, then push again; subsequent pushes to the same branch automatically trigger a rerun, with no need to reopen the PR. For review feedback, just add follow-up commits on the branch.

## Documentation changes must keep Chinese and English in sync

Each documentation book (`docs/user-guide` and `docs/dev-guide`) keeps two independent sources: `src/` is Simplified Chinese (the source language) and `en/` is English. The two are independent Markdown files with no automatic translation, so when you change the Chinese you must update the English by hand.

Small changes such as fixing typos or adjusting wording may not affect the English; but as soon as you add or remove a paragraph, or change a complete sentence, update the corresponding part of the same-named file under `en/` too, otherwise the English site stays on the old content. When adding or removing a chapter, remember to change both the chapter file and the respective `SUMMARY.md` under both `src/` and `en/`.

Code blocks are written separately in the two sources: the Chinese version in `src/` uses Chinese in its comments and example output, while the English version in `en/` uses English; commands, paths, and identifiers stay identical on both sides. When you are done, build both languages locally and confirm each renders correctly (for English, use `MDBOOK_BOOK__SRC=en` to switch the source dir to `en/`):

```bash
cd docs/dev-guide
mdbook build                                              # Chinese (src/)
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook build  # English (en/)
```

For the directory layout, the language switcher, and the CI build details, see [documentation.md](documentation.md) and `.github/workflows/pages.yml`. Submit the Chinese source and the English changes in the same PR, so the reviewer can read them side by side and the English site gets updated along with the same merge.

## Running into problems

If you're unsure how to make a change, or you'd like to discuss the direction first, you're welcome to open an issue to talk it over before you start writing code. Small fixes can go straight to a PR. Either way, we're happy to help you move your change toward a merge.
