# cjv for Cangjie Central Repository

This package publishes `cjv` to the Cangjie central repository as a permanent
source-only shim. It carries no prebuilt binary and is intentionally pinned to
a single version (`1.0.0`); republishing on each `cjv` release is unnecessary.

`cjpm install --path <module>` or `cjpm install cjv-1.0.0` compiles a
placeholder executable, then the build script marks the install flow during
`pre-install` and replaces `target/release/bin/main` with the latest prebuilt
`cjv` binary fetched from GitHub Releases during `post-build`.

The shim resolves the latest tag via the GitHub API at install time. Set
`CJV_VERSION=v0.2.0` (or just `0.2.0`) in the environment to pin a specific
release instead.

Notes:

- The package intentionally ships source files only.
- v1 supports the default `cjpm` install root only.
- The real executable is downloaded only as part of `cjpm install`; a plain
  `cjpm build` keeps the placeholder binary.
