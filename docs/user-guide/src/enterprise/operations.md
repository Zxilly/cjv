# 发布、升级与验收

## cjv 本体升级

工具链 `dist_server` 不控制 cjv 本体更新。企业环境建议：

1. 在受控环境获取 cjv 发布归档和同一发布的 `checksums.txt`。
2. 校验后发布到企业制品库或软件分发系统。
3. 在客户端设置 `auto_self_update = "disable"`。
4. 通过 GPO、终端管理平台、系统包或基础镜像统一升级 cjv。

`CJV_UPDATE_ROOT` 可供安装脚本首次下载内部归档，但不会持久化为已安装 cjv 的更新源。

## Nightly 发布策略

nightly 在统一 manifest 中是普通通道，但发布策略应比 LTS/STS 更严格：

- 先镜像同一版本的全部批准平台、目标 SDK 和组件，再推进 `channels.nightly.latest`。
- `latest` 指向的版本如果缺少请求的目标或组件，cjv 会失败，不会自动选择更早的 nightly。
- 保留所有被 `cangjie-sdk.toml` 固定的 nightly 版本及其组件。
- 上游 Release tag 与 SDK 版本不一致时，在 SDK 条目写入 `release_tag`；不要靠客户端猜测命名。
- manifest 更新应原子发布，避免客户端读到半写入内容。

## 上线验收清单

- 从普通用户会话执行 `cjv toolchain list-remote --channel lts` 和 `--channel nightly`，确认两者都只命中 `<dist_server>/versions.json`。
- 在未配置 `CJV_GITCODE_API_KEY` 的终端安装批准的 nightly，确认安装成功。
- 安装批准版本及所需组件后，执行 `cjv which cjc` 与 `cjc --version`。
- 检查 manifest 中 host SDK、交叉编译目标、stdx、docs 和 stdx-docs 的 URL，确认都符合企业批准的访问范围。
- 临时从 manifest 删除 nightly 或某个组件，确认命令直接失败且没有访问 GitHub、GitCode 或其他上游地址。
- 断开网络后再次编译，确认已安装工具链不产生下载请求。
- 确认 `auto_install = false`、`auto_self_update = "disable"`，并由企业流程负责升级。
- 验证企业 CA、代理变量和 `NO_PROXY` 在实际终端与 CI 服务账户下均生效。
- 用网络 ACL 验证终端无法绕过内部分发源访问公网下载地址。

系统后备配置用于提供一致默认值；真正的强制策略由网络 ACL、终端权限和软件分发流程共同完成。
