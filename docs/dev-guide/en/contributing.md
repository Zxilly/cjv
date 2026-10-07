# Contributing

Create a branch from the latest `master` and keep the change focused. External contributors can fork [Zxilly/cjv](https://github.com/Zxilly/cjv) and open a PR against the upstream `master` branch.

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

## Changes and validation

Reproduce the problem before fixing it in the module that owns the behavior. Test results through production entry points; see [testing and checks](testing.md) for locations and commands. For platform-specific behavior, state which platforms you tested and let CI check the others.

Keep Chinese and English documentation changes in the same commit. Chapter additions and removals also need contents updates and old-page redirects. Frontend copy uses Lingui and needs English translations.

## Commit messages

Before every commit, inspect recent history and follow the repository's current style:

```bash
git log --oneline -20
```

The current form is `type(scope): description`, with an optional scope and a concise English description. Examples:

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

Separate commits by responsibility. Keep unrelated formatting and dependency upgrades out of fixes.

## Pull requests

Describe the trigger, resulting behavior, and validation in the PR. Link related issues. Check CI after pushing, inspect failures, and add fixes to the same branch as review continues.

Update the user guide for command or configuration changes. Architecture documentation should explain stable responsibilities and constraints; implementation details belong in code and tests.
