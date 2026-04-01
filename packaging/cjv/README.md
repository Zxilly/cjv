# cjv for Cangjie Central Repository

This package publishes `cjv` to the Cangjie central repository as a source package.

`cjpm install --path <module>` or `cjpm install cjv-X.Y.Z` compiles a placeholder executable, then the build script marks the install flow during `pre-install` and replaces `target/release/bin/main` with the matching prebuilt `cjv` binary from GitHub Releases during `post-build`.

Notes:

- The package intentionally ships source files only. The real executable is downloaded only as part of `cjpm install`; a plain `cjpm build` keeps the placeholder binary.
- v1 supports the default `cjpm` install root only.
- The release workflow injects the final version into `cjpm.toml` and `release.env` before bundling and publishing.
