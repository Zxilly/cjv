A project that builds locally can fail in CI when the compiler differs. Commit the SDK declaration with the project and make the workflow read it to reduce differences between developer machines and build runners.

The following template targets an Ubuntu GitHub-hosted runner. It assumes a buildable, testable cjpm project at the repository root and a `cangjie-sdk.toml` pinned to a specific release. The SDK must support the runner's operating system and architecture. Validate the workflow and dependencies in your own repository.

## Commit the project toolchain first

Run these commands locally at the project root:

```bash
cjv pin lts
cjv pin --check
```

Replace `lts` if the project requires another channel or a particular release. Inspect and commit the generated file. `cjv pin --check` does not modify it or access the network, making it useful for rejecting floating versions in CI. See [pinning project versions](https://cjv.zxilly.com/en/blog/pin-cangjie-project-versions/).

## Add a build and test workflow

Save this as `.github/workflows/cangjie.yml`:

```yaml
name: Cangjie

on:
  push:
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - name: Install cjv
        shell: bash
        run: |
          curl -sSf https://cjv.zxilly.com/install.sh -o "$RUNNER_TEMP/install-cjv.sh"
          sh "$RUNNER_TEMP/install-cjv.sh" -y --default-toolchain none
          echo "$HOME/.cjv/bin" >> "$GITHUB_PATH"
      - name: Validate and prepare the project SDK
        shell: bash
        run: |
          cjv --version
          cjv pin --check
          cjc --version
          cjv show active
      - name: Build and test
        shell: bash
        run: |
          cjpm build
          cjpm test
```

`--default-toolchain none` installs only the manager. The subsequent `cjc --version` invocation selects the project's SDK and prepares missing requirements with automatic installation enabled by default. `GITHUB_PATH` applies to later steps, so installation and compiler invocation are separate steps.

This template downloads the current stable cjv release. It pins the project's SDK, not the manager itself. For stricter tool reproducibility, add a project-validated release tag through the installer's `--version` option and maintain it under your update policy. A fixed runner label does not freeze every package in the runner image either.

## Define what the cache contains

First get an uncached workflow working. Then consider caching `<CJV_HOME>/cache`, which defaults to `~/.cjv/cache`. It holds reusable manifests and verified download archives, separately from installed SDK directories.

When using `actions/cache`, include the runner OS, architecture, project toolchain file hash and a cache-format version in the key. Distinguish distribution sources when changing them. Do not share a complete installed SDK directory across platforms.

A cache hit does not guarantee an offline build: components, cross targets or cjpm dependencies may still be needed. `cjv --offline` restricts cjv's requests, not network access by its child processes. See the [automation guide](https://cjv.zxilly.com/book/user-guide/en/automation.html) for preparing offline environments.

## Diagnose failures by stage

- **cjv or cjc is not found:** confirm that installation succeeded and the previous step wrote the correct directory to `GITHUB_PATH`.
- **Pin validation fails:** check that the current directory contains a `cangjie-sdk.toml` with a specific release. For a monorepo, give validation and build steps the same `working-directory`.
- **An SDK or component download fails:** inspect the error and confirm the release exists for the selected platform. Never reuse an archive whose checksum failed as a valid cache entry.
- **The compiler starts but a build or test fails:** inspect cjpm logs, system dependencies and project code. Successful SDK installation does not establish that tests pass.

For self-hosted runners, also isolate each job's `CJV_HOME` and settings file to avoid inheriting another project's overrides or environment variables. See [configuration and networking](https://cjv.zxilly.com/book/user-guide/en/automation.html).
