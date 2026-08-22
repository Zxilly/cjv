# 通道

通道（channel）是 cjv 对仓颉 SDK 发布流的命名。每个通道代表一条持续更新的发布线。安装一个通道时，cjv 会从版本清单解析该通道当前的最新版本并安装它。

| 通道 | 含义 | 元数据来源 |
| --- | --- | --- |
| `lts` | 长期支持版 | 版本清单（manifest） |
| `sts` | 短期支持版 | 版本清单（manifest） |
| `nightly` | 每日构建（预览版） | 版本清单（manifest） |

通道名大小写不敏感，`LTS`、`Lts`、`lts` 等价。

## 选哪个通道

`lts` 版本相对稳定、维护周期长，适合生产构建以及对兼容性敏感的项目。`sts` 更新更快，适合希望较早使用新特性的项目。`nightly` 包含最新但尚未稳定的改动，适合尝鲜、复现 upstream 行为或为 SDK 本体提 bug。

```bash
cjv install lts
cjv install sts
cjv install nightly
```

## 通道与版本名

把通道名交给 `cjv install` 会安装该通道的最新版本，并以 `<通道>-<版本>` 的名称落盘。也可以固定具体版本：

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001

# 裸版本号在 LTS / STS 中查找所属通道
cjv install 1.0.5
```

nightly 的具体版本使用带通道前缀的名称。完整命名规则见[工具链](toolchains.md)。

## Manifest 分发模型

三个通道共用一份 JSON 版本清单。manifest 记录可用版本、每个平台的 SDK URL、组件 URL 与校验和；cjv 先解析清单，再下载其中指定的制品。

默认清单由 [`cangjie-version-manifest`](https://github.com/Zxilly/cangjie-version-manifest) 维护。正式版本更新通过 PR 审核后合入，nightly 由定时任务采集 GitCode `Cangjie/nightly_build` Release 并直接更新清单。客户端读取生成后的静态清单。

清单地址可在 `~/.cjv/settings.toml` 中通过 `manifest_url` 覆盖。配置 `dist_server` 或 `CJV_DIST_SERVER` 时，cjv 读取 `<dist_server>/versions.json`，适合企业统一托管。来源优先级和部署契约见[配置](../configuration.md)与[内部分发源](../enterprise/distribution-server.md)。

## 通道与组件

manifest 记录 stdx、docs、stdx-docs 的实际下载 URL。默认清单采集的上游来源如下：

| 组件 | LTS / STS 来源 | nightly 来源 |
| --- | --- | --- |
| `stdx` | `cangjie_stdx` 发布 | `nightly_build` 发布 |
| `docs` | `cangjie-docs-bundle` 发布 | `nightly_build` 发布 |
| `stdx-docs` | `cangjie_stdx` 发布 | `nightly_build` 发布 |

三个通道的组件都使用 manifest 中声明的 URL，并可携带 SHA-256。组件机制见[组件](components.md)。

```bash
cjv install nightly -c stdx,docs
```

## 在工具链文件中指定通道

项目可以在 `cangjie-sdk.toml` 中声明通道，让协作者使用相同的工具链选择：

```toml
[toolchain]
channel = "lts"
```

`channel` 可以是通道名，也可以是带版本的工具链名。完整字段语义见[工具链文件](../toolchain-file.md)。

## 检查更新

`cjv check` 查询 manifest，并比较已安装通道与各通道的最新版本：

```bash
cjv check
```
