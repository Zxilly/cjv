# 企业部署概览

企业网络可以让 cjv 通过统一代理访问上游，也可以把工具链和组件发布到内部 HTTPS 制品库。网络请求集中在安装、更新、远程查询和自动补齐阶段；已安装工具链的日常命令在本地执行。

## 选择部署模式

| 网络条件 | 推荐模式 | 支持范围 |
| --- | --- | --- |
| 允许经企业代理访问公网 | 设置标准代理环境变量 | 全部通道和自更新，受上游可用性约束 |
| 终端只能访问内部制品库 | 配置统一 `dist_server` | LTS、STS、nightly、SDK 与组件均可完整内网化 |
| 完全离线或气隙环境 | 本地归档或目录链接 | 已安装或本地提供的工具链可用 |
| 要求强制统一源和版本 | 内部分发源 + 网络 ACL + 企业软件分发 | 系统后备配置提供默认值，网络 ACL 实施访问策略 |

`mirror` 构建变体提供 GitCode 默认端点，适合 GitHub 访问不稳定的网络。企业内部镜像使用 `dist_server`。

## 工具链分发与 cjv 更新是两条链路

cjv 将两类制品分开处理：

- **工具链分发源**：SDK、stdx、docs、stdx-docs，以及 LTS、STS、nightly 的版本元数据。企业部署通过 `dist_server` 或 `CJV_DIST_SERVER` 统一控制。
- **cjv 本体**：安装脚本首次下载 cjv 时可使用 `CJV_UPDATE_ROOT`。企业软件分发系统负责已安装 cjv 的升级，客户端配置 `auto_self_update = "disable"`。

配置 `dist_server` 后，cjv 从 `<dist_server>/versions.json` 按需读取 LTS/STS，从 `<dist_server>/nightly.json` 按需读取 nightly。两份文件分别携带对应通道的 SDK 与组件。

## 网络访问范围

| 操作 | 默认来源 | 企业替代方式 |
| --- | --- | --- |
| 安装 cjv 本体 | GitHub 或 GitCode Release | 内部托管发布归档与 `checksums.txt`，安装脚本设置 `CJV_UPDATE_ROOT` |
| 安装、查询或更新工具链 | 默认 manifest | 配置 `dist_server` |
| 下载 SDK 和组件 | manifest 中声明的 URL | 在对应的 `versions.json` 或 `nightly.json` 中声明批准的 URL |
| `cjv self update` | 官方版使用 GitHub，mirror 版使用 GitCode | 关闭自更新，由企业软件分发系统升级 cjv |

`cjpm` 项目依赖仓库和凭据遵循项目自身配置；企业还需配置对应的依赖仓库镜像。cjv 的企业分发源覆盖 SDK 与组件。

接下来依次完成：

1. [建设内部分发源](distribution-server.md)
2. [部署受管客户端](client-deployment.md)
3. [配置代理或完全离线终端](restricted-networks.md)
4. [制定发布、升级与验收流程](operations.md)
