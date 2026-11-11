A maintenance project may need a stable SDK while a newer project uses another release. Selecting the toolchain from the project directory lets you switch work without rewriting the global environment each time.

The example uses two existing Cangjie projects in sibling directories, `project-a` and `project-b`, with cjv already installed. LTS and STS must be available for your host. See the [command reference](https://cjv.zxilly.com/book/user-guide/en/command-reference.html) for the supported `cjv pin` options.

## Pin each project's version

Inside the first project, resolve LTS to a specific release and write the project declaration:

```bash
cd project-a
cjv pin lts
cjv pin --check
cjc --version
```

`cjv pin` writes `cangjie-sdk.toml` without installing an SDK. If the corresponding tracking channel is already installed, pinning prefers that installed release. With automatic installation enabled, the first compiler invocation installs a missing pinned SDK.

Move into the second project and select its required STS release:

```bash
cd ../project-b
cjv pin sts
cjv pin --check
cjc --version
```

Inspect the generated files: each should contain its resolved, complete release name. If the team already requires a specific version, replace `lts` or `sts` with that full name from the distribution. Commit each project's `cangjie-sdk.toml` to version control.

## Verify selection by changing directories

```bash
cd ../project-a
cjv show active
cjv which cjc
cjpm build

cd ../project-b
cjv show active
cjv which cjc
cjpm build
```

Check that `show active` reports the current project's toolchain and selection source, and that `which cjc` points into the expected SDK. The exact version depends on the release you pinned.

A pinned release and a tracking channel are separate installations. Even if `lts` is installed, the first use of its resolved pinned release may require installation. Updating a tracking channel does not rewrite the project file.

## Try another SDK temporarily

To test project-a with nightly without changing the team's committed configuration, select it for individual commands:

```bash
cjv exec +nightly cjc --version
cjv exec +nightly cjpm build
```

For several consecutive commands, use a local directory override:

```bash
cjv override set nightly
cjv show active
cjpm build
cjv override unset
cjv show active
```

The override is stored in local settings. Remove it after testing and check that selection returns to the project file. Nightly availability and compatibility depend on the distribution; preserve existing build outputs or use a separate working directory for experiments.

## Investigate an unexpected version

Start with `cjv show active`, then check explicit command-line selections and `CJV_TOOLCHAIN`. Both take precedence over project files.

Directory configuration is resolved from the nearest directory upward. Within one directory, a local override wins over its project file; a child directory's project file can win over an ancestor's override. Files from different levels are not merged. See [project toolchains](https://cjv.zxilly.com/book/user-guide/en/toolchain-file.html) for the full rules.

Declare required stdx components and cross targets in the project file too. An explicit selection of another toolchain does not inherit the project's component and target requirements, so prepare those dependencies in the test environment as well.

Finally, run `cjv pin --check` in CI to catch floating channel declarations before building. Continue with the [GitHub Actions guide](https://cjv.zxilly.com/en/blog/cangjie-github-actions/).
