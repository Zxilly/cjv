# cjv for Cangjie Central Repository

This package publishes `cjv` to the Cangjie central repository as a source package.

`cjpm install --path <module>` or `cjpm install cjv-X.Y.Z` compiles a placeholder executable, then the build script downloads the matching prebuilt `cjv` binary from GitHub Releases during `post-install` and places it under the default `cjpm` binary directory.

Notes:

- The package intentionally ships source files only. The real executable is downloaded during `post-install`, not during a plain `cjpm build`.
- v1 supports the default `cjpm` install root only.
- The release workflow injects the final version into `cjpm.toml` and `release.env` before bundling and publishing.
