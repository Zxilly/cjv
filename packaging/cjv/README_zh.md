# 仓颉中心仓 cjv 包

该模块用于把 `cjv` 作为源码包发布到仓颉中心仓。

执行 `cjpm install --path <module>` 或 `cjpm install cjv-X.Y.Z` 时，`cjpm` 会先编译占位可执行文件，构建脚本会在 `pre-install` 阶段标记安装流程，并在 `post-build` 阶段从 GitHub Releases 下载对应平台的 `cjv` 二进制，替换 `target/release/bin/main`。

说明：

- 包内只包含源码和脚本，不直接携带预编译二进制。真实二进制仅会在 `cjpm install` 流程中下载，单独执行 `cjpm build` 仍然保留占位产物。
- v1 仅支持 `cjpm` 默认安装根路径。
- 发布流水线会在打包前把真实版本号写入 `cjpm.toml` 和 `release.env`。
