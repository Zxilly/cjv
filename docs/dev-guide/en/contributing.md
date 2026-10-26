# Contributing

Contributions to `cjv`—whether bug reports, documentation improvements, or code changes—are very welcome! To get started, create a focused branch from the latest `master`. External contributors can fork [Zxilly/cjv](https://github.com/Zxilly/cjv) on GitHub and open a pull request against the upstream `master` branch:

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

## Making changes and validating locally

1. **Reproduce first** — Before fixing a bug, reproduce the failure and locate the package that owns the behavior in the [Architecture guide](architecture.md).
2. **Test through production entry points** — Add or update regression tests in the owning package rather than testing private helpers in isolation. See [Testing and checks](testing.md#where-regression-tests-belong) for test locations and commands.
3. **Note platform coverage** — When changing platform-specific behavior (such as Windows file locking, Unix shell profile edits, or OpenHarmony self-signing), mention which platforms you tested locally in your PR description and let CI verify the rest of the matrix.
4. **Keep translations and docs in sync** — Keep Chinese (`src/`) and English (`en/`) documentation changes in the same commit. When adding, moving, or removing a chapter, update both `SUMMARY.md` files and add a redirect entry in `book.toml` (see [Documentation site](documentation.md)). Frontend copy in `web/` uses Lingui (`pnpm i18n:extract`) and requires English translations in `web/src/locales/en/messages.po` (see [Landing page](web.md#translation-and-styling)).

## Atomic commits and commit messages

We prefer atomic commits where each commit represents a single, self-contained logical change. Keep unrelated formatting cleanups, refactoring, and dependency upgrades in separate commits from behavioral fixes.

Before committing, inspect the recent git history and follow the repository's convention:

```bash
git log --oneline -20
```

Commit titles use the Conventional Commits form `type(scope): description`, where `(scope)` is optional and `description` is a concise imperative phrase in English:

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

## Pull requests

When opening a pull request, describe:

- **The trigger or motivation** — What user command, configuration, or edge case triggered the problem.
- **The resulting behavior** — How `cjv` behaves after your change.
- **Validation** — Which test suites and platforms you ran locally.
- **Related issues** — Links or `#issue` references to any related GitHub issues.

After pushing your branch, check the GitHub Actions results, inspect any failures, and push follow-up fixes to the same branch as code review continues.

Remember to update the [User Guide](https://cjv.zxilly.com/book/user-guide/en/) whenever a change alters CLI flags, configuration fields, or user-visible behavior. In the Developer Guide, keep [Architecture](architecture.md) focused on stable module responsibilities and invariants—low-level implementation details belong in code comments and tests.
