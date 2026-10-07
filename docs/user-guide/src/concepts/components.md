# 组件与离线文档

组件按工具链分别安装，卸载工具链时一起清理：

| 组件 | 内容 | 相对 `CJV_HOME` 的位置 |
| --- | --- | --- |
| `stdx` | 扩展库 | `stdx/<tc>/{dynamic,static}` |
| `docs` | 语言、标准库和工具文档 | `docs/<tc>/main/` |
| `stdx-docs` | 扩展库文档 | `docs/<tc>/stdx/` |

## 安装与移除

```bash
cjv install nightly -c stdx,docs
cjv component add stdx docs --toolchain lts
cjv component list --toolchain lts
cjv component list --installed -q
cjv component remove stdx-docs
```

未指定 `--toolchain` 时使用当前选择；`--target <suffix>` 选择该宿主已安装的交叉 SDK。组件名可重复或逗号分隔。`component add` 跳过已安装项，加 `--force` 可重新安装。

可用制品取决于所选通道、版本和平台。自定义工具链没有官方组件下载源。项目也可在 [cangjie-sdk.toml](../toolchain-file.md) 中声明所需组件，由自动安装补齐。

跟踪通道升级时，SDK 和所选组件一起更新，固定版本的组件不变。本地链接的 stdx 保留原来源。

## 链接本地 stdx

```bash
cjv component link stdx /path/to/local/stdx --toolchain lts --force
```

源目录必须包含 `dynamic/` 和 `static/`。`--force` 替换已有组件；移除链接只删除 cjv 管理的条目，保留源目录。`docs` 和 `stdx-docs` 不支持本地链接。

组件修改要求 SDK 位于 cjv 管理的真实目录中。通过 `toolchain link` 引用的外部 SDK 目录不支持组件修改；需要 cjv 管理组件时，先从[本地归档或 URL](../install-from-url.md)安装 SDK：

```bash
cjv toolchain link my-sdk ./cangjie-sdk.zip
cjv component link stdx /path/to/local/stdx --toolchain my-sdk
```

stdx 安装后，代理、`exec` 和 `envsetup` 会提供 `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`。文档组件不修改运行环境。

## 打开文档

```bash
cjv doc
cjv doc std
cjv doc dev-guide
cjv doc tools
cjv doc stdx --toolchain nightly
cjv doc --path
```

`doc` 在浏览器中打开本地 HTML；`--path` 或 `--json` 只返回路径。`book` 是 `dev-guide` 的别名。不带主题时优先打开主体文档，否则尝试扩展库文档；缺少文档时会提示安装对应组件。
