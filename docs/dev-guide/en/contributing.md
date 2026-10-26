# Contribute to cjv

We welcome bug reports, feature requests, and pull requests. Before you start working on a change, read [Build from source](building.md) and [Architecture](architecture.md) to find the package that owns the behavior.

## Create a working branch

Create a focused branch from the latest `master` branch. If you are an external contributor, fork [Zxilly/cjv](https://github.com/Zxilly/cjv) on GitHub and branch from `upstream/master`:

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

Keep each branch focused on a single bug fix or feature. Do not mix unrelated formatting changes or dependency upgrades into a functional fix.

## Implement and validate changes

### Code and test requirements

1. Reproduce the bug or define the target behavior before editing production code.
2. Implement the change in the package that owns that responsibility; read [Architecture](architecture.md).
3. Add or update unit and integration tests through the production entry points; read [Tests and checks](testing.md#where-to-place-regression-tests).
4. If your change affects platform-specific behavior (such as Windows file locks, Unix shell profiles, or OpenHarmony signing), note which platforms you tested locally and let CI check the full platform matrix.

### Documentation and localization synchronization

- **Bilingual manuals**: Whenever you change CLI flags, configuration fields, or user-visible behavior, update both the Chinese (`src/`) and English (`en/`) mdBook files in the same commit.
- **Chapter changes**: If you add, move, or remove a documentation page, update both `SUMMARY.md` files and add a redirect rule under `[output.html.redirect]` in `book.toml`. Read [Documentation site](documentation.md).
- **Frontend copy**: If you add or edit text in `web/`, extract the Lingui catalog (`pnpm i18n:extract`) and translate the new strings in `web/src/locales/en/messages.po`. Read [Landing page](web.md#translation-and-styling).

## Write commit messages

Check the recent commit log to match the repository's convention:

```bash
git log --oneline -20
```

Write commit titles in the format `type(scope): description`, where `(scope)` is optional and `description` is a short imperative phrase in English:

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

Split independent logical changes into separate commits.

## Open a pull request

In your pull request description, include:

- **Problem or trigger**: what user command or condition triggered the issue.
- **Solution and behavior**: what changed and how `cjv` behaves now.
- **Validation**: which test suites and platforms you ran locally (run `npx markdownlint-cli2` and `python3 scripts/build-docs.py` for documentation changes).
- **Related issues**: links to any GitHub issues that the PR addresses.

After pushing your branch, check the GitHub Actions results. If a CI job fails, inspect the log and push follow-up commits to the same branch. Keep architecture docs focused on stable package contracts and invariants; put low-level implementation details in code comments and tests.
