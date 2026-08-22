# 发布、升级与验收

## cjv 本体升级

工具链由 `dist_server` 分发，cjv 本体由企业软件分发系统升级。建议流程如下：

1. 在受控环境获取 cjv 发布归档和同一发布的 `checksums.txt`。
2. 校验后发布到企业制品库或软件分发系统。
3. 在客户端设置 `auto_self_update = "disable"`。
4. 通过 GPO、终端管理平台、系统包或基础镜像统一升级 cjv。

`CJV_UPDATE_ROOT` 为安装脚本选择首次下载地址；后续版本由上述企业升级流程发布。

## Nightly 发布策略

nightly 在统一 manifest 中是普通通道，但发布策略应比 LTS/STS 更严格：

- 先镜像同一版本的全部批准平台、目标 SDK 和组件，再推进 `channels.nightly.latest`。
- `latest` 精确选择一个版本；缺少目标或组件时返回对应错误。
- 保留所有被 `cangjie-sdk.toml` 固定的 nightly 版本及其组件。
- 上游 Release tag 与 SDK 版本存在差异时，在 SDK 条目写入 `release_tag` 作为权威映射。
- manifest 使用原子发布，让客户端始终读取完整版本。

## 上线验收清单

- 从普通用户会话执行 `cjv toolchain list-remote --channel lts` 和 `--channel nightly`，确认两者都只命中 `<dist_server>/versions.json`。
- 在仅配置 `dist_server` 的终端安装批准的 nightly，确认安装成功。
- 安装批准版本及所需组件后，执行 `cjv which cjc` 与 `cjc --version`。
- 检查 manifest 中 host SDK、交叉编译目标、stdx、docs 和 stdx-docs 的 URL，确认都符合企业批准的访问范围。
- 临时从 manifest 删除 nightly 或某个组件，确认命令返回明确的缺失错误，并核对分发端访问日志。
- 断开网络后再次编译，确认已安装工具链完成本地构建。
- 确认 `auto_install = false`、`auto_self_update = "disable"`，并由企业流程负责升级。
- 验证企业 CA、代理变量和 `NO_PROXY` 在实际终端与 CI 服务账户下均生效。
- 用网络 ACL 将终端访问范围限制为企业批准的下载地址。

系统后备配置用于提供一致默认值；真正的强制策略由网络 ACL、终端权限和软件分发流程共同完成。
