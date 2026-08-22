# 企业部署概览

企业网络可以让 cjv 通过统一代理访问上游，也可以把工具链和组件发布到内部 HTTPS 制品库。工具链安装完成后，日常执行 `cjc`、`cjpm` 等命令不需要网络；安装、更新、远程查询或自动补齐缺失内容时才会发起请求。

## 选择部署模式

| 网络条件 | 推荐模式 | 支持范围 |
| --- | --- | --- |
| 允许经企业代理访问公网 | 设置标准代理环境变量 | 全部通道和自更新，受上游可用性约束 |
| 终端只能访问内部制品库 | 配置统一 `dist_server` | LTS、STS、nightly、SDK 与组件均可完整内网化 |
| 完全离线或气隙环境 | 本地归档或目录链接 | 已安装或本地提供的工具链可用 |
| 要求强制统一源和版本 | 内部分发源 + 网络 ACL + 企业软件分发 | cjv 的系统后备配置本身不是强制策略 |

`mirror` 构建变体只把默认 LTS/STS manifest 和 cjv 自更新后端从 GitHub 切换到 GitCode，适合 GitHub 访问不稳定的网络；它不是通用的企业内部镜像版。

## 工具链分发与 cjv 更新是两条链路

cjv 将两类制品分开处理：

- **工具链分发源**：SDK、stdx、docs、stdx-docs，以及 LTS、STS、nightly 的版本元数据。企业部署通过 `dist_server` 或 `CJV_DIST_SERVER` 统一控制。
- **cjv 本体**：安装脚本首次下载 cjv 时可使用 `CJV_UPDATE_ROOT`。已安装 cjv 的升级建议交给企业软件分发系统，并关闭 cjv 自更新。

配置 `dist_server` 后，cjv 会读取 `<dist_server>/versions.json`，所有通道和组件都必须由这份 manifest 描述。缺少 nightly 或某个制品时会直接报错，不会回退到 GitCode 或其他公网地址。

未配置 `dist_server` 时保留兼容行为：LTS/STS 使用 `manifest_url`，nightly 使用 GitCode `nightly_build` Release。

## 网络访问范围

| 操作 | 默认来源 | 企业替代方式 |
| --- | --- | --- |
| 安装 cjv 本体 | GitHub 或 GitCode Release | 内部托管发布归档与 `checksums.txt`，安装脚本设置 `CJV_UPDATE_ROOT` |
| 安装、查询或更新工具链 | LTS/STS manifest；nightly GitCode API | 配置统一 `dist_server` |
| 下载 SDK 和组件 | manifest URL 或 nightly Release | 在 `<dist_server>/versions.json` 中声明批准的相对或绝对 URL |
| `cjv self update` | 官方版使用 GitHub，mirror 版使用 GitCode | 关闭自更新，由企业软件分发系统升级 cjv |

`cjpm` 下载项目依赖时使用的仓库和凭据不由 cjv 管理。部署 cjv 只解决 SDK 与组件分发，依赖仓库镜像需要单独配置。

接下来依次完成：

1. [建设内部分发源](distribution-server.md)
2. [部署受管客户端](client-deployment.md)
3. [配置代理或完全离线终端](restricted-networks.md)
4. [制定发布、升级与验收流程](operations.md)
