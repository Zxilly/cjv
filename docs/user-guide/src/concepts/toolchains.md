# 工具链与版本

一个工具链是一份仓颉 SDK，包含编译器、构建工具和运行时库。cjv 可以同时保存多份 SDK，安装在 `<CJV_HOME>/toolchains/` 下；组件和离线文档另行存放。

## 通道与名称

| 名称 | 含义 |
| --- | --- |
| `lts` | 跟踪长期支持通道 |
| `sts` | 跟踪短期支持通道 |
| `nightly` | 跟踪开发构建通道，可用版本取决于已发布清单 |
| `lts-1.0.5` | 固定通道中的具体版本 |
| `1.0.5` | 省略通道，由 cjv 查找匹配版本 |
| `sts-1.2`、`1.2` | 选择该次版本中的最新稳定补丁 |
| `nightly-2026-10-04` | 选择该日期最新的已发布 nightly |
| `my-sdk` | 用 `cjv toolchain link` 创建的自定义工具链 |

表中的版本和日期仅用于展示名称格式。用以下命令查询当前分发源：

```bash
cjv toolchain list-remote --channel lts
cjv toolchain list-remote --channel nightly --limit 10
```

通道名不区分大小写。自定义名不能与官方名称冲突，不能包含路径分隔符或以 `+` 开头，也不能是 `.`、`..` 或空字符串。

## 跟踪通道与固定版本

```bash
cjv install lts
cjv install lts-1.0.5
```

这会创建两个独立安装。`cjv update lts` 更新跟踪通道，固定的 `lts-1.0.5` 保留原样；两者的组件也分别管理。运行 `cjc` 时使用已安装版本，更新需要执行 `install` 或 `update`。

次版本和日期简写安装为解析后的具体版本。需要团队使用相同版本时，在 [cangjie-sdk.toml](../toolchain-file.md) 中写入完整版本名。

nightly 安装会考虑所需组件和交叉 SDK，在清单保留的历史版本中选择兼容版本；默认不降级。`--allow-downgrade` 允许选择更旧的兼容 nightly。其他安装策略见[命令参考](../command-reference.md)。

## 主机平台

`sts-linux-x64`、`sts-1.2.0-darwin-arm64` 可以指定主机平台。不同主机平台独立安装，当前默认平台的显式名称与省略平台的名称指向同一安装。安装其他平台的 SDK 不会让该平台的程序自动具备本机运行能力。

交叉编译 SDK 通过[目标管理](../cross-compilation.md)附加到宿主工具链，不能设为默认或活跃工具链。

## 自定义 SDK

```bash
cjv toolchain link my-sdk /path/to/local/sdk
cjv default my-sdk
```

目录链接直接引用已有 SDK，源目录由你维护；卸载只移除链接。需要由 cjv 管理副本或添加本地 stdx 时，可改用[归档安装](../install-from-url.md)。自定义工具链不会通过 `cjv update` 更新，也没有可供 `component add` 下载的官方组件。

用 `cjv toolchain list` 查看安装，用 `cjv show active` 查看当前选择及来源。选择规则见[项目工具链](../toolchain-file.md)。
