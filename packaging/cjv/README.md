# cjv for Cangjie Central Repository

This package publishes `cjv` to the Cangjie central repository as a permanent
source-only shim. It carries no prebuilt binary and is intentionally pinned to
a single version (`1.0.0`); republishing on each `cjv` release is unnecessary.

`cjpm install --path <module>` or `cjpm install cjv-1.0.0` compiles a
placeholder executable, then the build script marks the install flow during
`pre-install` and replaces `target/release/bin/main` with the latest prebuilt
`cjv` binary fetched from GitHub Releases during `post-build`.

The shim resolves the latest tag via the GitHub API at install time. The
following environment variables override the defaults:

- `CJV_VERSION` — pin a specific tag, e.g. `v0.2.0` or `0.2.0`.
- `CJV_REPOSITORY` — `<owner>/<repo>` hosting the cjv binary releases (default `Zxilly/cjv`).
- `CJV_RELEASE_BASE_URL` — asset download base URL (default `https://github.com/<repo>/releases/download`).
- `CJV_API_BASE_URL` — GitHub API base URL used for the latest-tag lookup (default `https://api.github.com`).

Notes:

- The package intentionally ships source files only.
- v1 supports the default `cjpm` install root only.
- The real executable is downloaded only as part of `cjpm install`; a plain
  `cjpm build` keeps the placeholder binary.
- The shim is not republished by CI. To bootstrap a fresh repository or push
  an updated shim, run `cjpm bundle && cjpm publish` from this directory
  locally with `~/.cjpm/cangjie-repo.toml` configured.
