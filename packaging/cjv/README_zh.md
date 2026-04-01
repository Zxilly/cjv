# 仓颉中心仓 cjv 包

该模块用于把 `cjv` 作为源码包发布到仓颉中心仓。

执行 `cjpm install --path <module>` 或 `cjpm install cjv-X.Y.Z` 时，`cjpm` 会先编译一个占位可执行文件，然后由构建脚本在 `post-build` / `post-install` 阶段从 GitHub Releases 下载对应平台的 `cjv` 二进制并替换占位产物。

说明：

- 包内只包含源码和脚本，不直接携带预编译二进制。
- v1 仅支持 `cjpm` 默认安装根路径。
- 发布流水线会在打包前把真实版本号写入 `cjpm.toml` 和 `release.env`。
