# 常见问题

## 为什么选中的版本不对？

运行 `cjv show active` 查看选择及来源，再检查 `CJV_TOOLCHAIN`、当前目录及父目录的覆盖和 `cangjie-sdk.toml`。cjv 逐层检查目录，同层覆盖优先；完整规则见[项目工具链](toolchain-file.md)。

如果执行过 `envsetup`，当前 shell 可能直接调用某个 SDK。打开新终端恢复代理选择，或重新加载所需工具链的环境。

## 项目文件拼错字段后会怎样？

未知字段会警告。如果剩余字段没有提供有效的 `channel` 或 `path`，命令会报错，不会退回默认工具链。空文件同样如此。需要恢复上级选择时删除文件。

## 编译产物提示找不到运行时库怎么办？

```bash
cjv exec ./my_binary
```

也可以为当前 shell 加载 `cjv envsetup` 的输出。各 shell 的写法见[运行环境](runtime-environment.md)。

## 离线环境怎样安装？

使用[本地 SDK 归档或目录](install-from-url.md)。已有工具链可本地运行，项目依赖仍须预先准备。受管终端可配置[内部分发源](enterprise/distribution-server.md)，并关闭自动安装和自更新检查，见[企业部署](enterprise/index.md)。

## 自定义工具链为什么不能下载组件？

自定义 SDK 没有官方发行记录，`component add` 无法确定对应制品。归档安装可携带 stdx，也可链接本地 stdx。外部目录链接由源 SDK 所有者维护，不支持 cjv 组件修改。参见[组件](concepts/components.md)。

## 卸载会删除什么？

`cjv uninstall <name>` 删除所选 SDK、附属 stdx 和文档，并清理引用它的目录覆盖。如果它是默认工具链，cjv 会尝试选用其他已安装宿主，否则清除默认选择。链接来源保留。

`cjv self uninstall` 删除 cjv 数据目录及所有安装，并清理 PATH。命令选项见[命令参考](command-reference.md)。

## macOS 下载页为什么让我选架构？

部分浏览器不能可靠报告 Mac 的 CPU 架构。安装脚本会在本机检测；手动下载可运行 `uname -m`，`arm64` 对应 Apple Silicon，`x86_64` 对应 Intel。

## nightly 缺少校验和时怎么处理？

cjv 从 `nightly.json` 读取版本和下载地址。SDK 条目没有 SHA-256 时会尝试 `<url>.sha256`；sidecar 尚未发布时会提示完整性限制。企业镜像可直接在清单中填写校验和，见[分发源格式](enterprise/distribution-server.md)。
