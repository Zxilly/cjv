# 仓颉中心仓 cjv 包

该模块用于把 `cjv` 作为永久 shim 发布到仓颉中心仓。包内不携带任何预编译二进制，
版本号固定为 `1.0.0`，每次发布新的 `cjv` 二进制不需要重新 publish。

执行 `cjpm install --path <module>` 或 `cjpm install cjv-1.0.0` 时，`cjpm` 会
先编译占位可执行文件，构建脚本会在 `pre-install` 阶段标记安装流程，并在
`post-build` 阶段从 GitHub Releases 下载最新版本的 `cjv` 二进制，替换
`target/release/bin/main`。

shim 在安装时通过 GitHub API 查询最新 tag。以下环境变量可覆盖默认值：

- `CJV_VERSION` —— 锁定到指定 tag，例如 `v0.2.0` 或 `0.2.0`。
- `CJV_REPOSITORY` —— 二进制 release 所在的 `<owner>/<repo>`（默认 `Zxilly/cjv`）。
- `CJV_RELEASE_BASE_URL` —— asset 下载基地址（默认 `https://github.com/<repo>/releases/download`）。
- `CJV_API_BASE_URL` —— 用于查询 latest tag 的 GitHub API 基地址（默认 `https://api.github.com`）。

说明：

- 包内只包含源码和脚本，不直接携带预编译二进制。
- v1 仅支持 `cjpm` 默认安装根路径。
- 真实二进制仅会在 `cjpm install` 流程中下载，单独执行 `cjpm build` 仍然保留
  占位产物。
- shim 不再由 CI 自动 publish。需要 bootstrap 新仓库或推送 shim 更新时，请在
  本目录下本地执行 `cjpm bundle && cjpm publish`，并提前配置好
  `~/.cjpm/cangjie-repo.toml`。
