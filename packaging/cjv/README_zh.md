# 仓颉中心仓 cjv 包

该模块用于把 `cjv` 作为源码包发布到仓颉中心仓。

执行 `cjpm install --path <module>` 或 `cjpm install cjv-X.Y.Z` 时，`cjpm` 会先编译占位可执行文件，随后在 `post-install` 阶段由构建脚本从 GitHub Releases 下载对应平台的 `cjv` 二进制并安装到 `cjpm` 默认二进制目录。仅执行 `cjpm build` 不会下载真实二进制。

说明：

- 包内只包含源码和脚本，不直接携带预编译二进制。
- v1 仅支持 `cjpm` 默认安装根路径。
- 发布流水线会在打包前把真实版本号写入 `cjpm.toml` 和 `release.env`。
