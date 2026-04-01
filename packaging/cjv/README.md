# cjv for Cangjie Central Repository

This package publishes `cjv` to the Cangjie central repository as a source package.

`cjpm install --path <module>` or `cjpm install cjv-X.Y.Z` compiles a placeholder executable, then the build script replaces it with the matching prebuilt `cjv` binary from GitHub Releases.

Notes:

- The package intentionally ships source files only. The real executable is downloaded during `post-build` and `post-install`.
- v1 supports the default `cjpm` install root only.
- The release workflow injects the final version into `cjpm.toml` and `release.env` before bundling and publishing.
