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

把通道名交给 `cjv install` 会安装该通道的最新版本，并以通道身份落盘，例如 `toolchains/sts`。具体版本存放在独立目录，例如 `toolchains/sts-1.2.0`；两个安装各自管理组件。也可以固定具体版本：

```bash
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23
cjv install nightly-1.1.0-alpha.20260306010001

# 裸版本号在 LTS / STS 中查找所属通道
cjv install 1.0.5
```

nightly 的具体版本使用带通道前缀的名称。完整命名规则见[工具链](toolchains.md)。

## Manifest 分发模型

正式通道与 nightly 使用独立的静态清单：`versions.json` 记录 LTS/STS，`nightly.json` 记录 nightly。两份文件都包含可用版本、平台 SDK URL、组件 URL 与校验和。cjv 按请求通道加载对应文件，因此普通 LTS/STS 操作无需下载 nightly 历史。

默认清单由 [`cangjie-version-manifest`](https://github.com/Zxilly/cangjie-version-manifest) 维护。正式版本通过 PR 更新 `versions.json`，nightly 定时任务采集 GitCode `Cangjie/nightly_build` Release 并直接更新 `nightly.json`。

`manifest_url` 指向正式通道文件，nightly 文件从同目录的 `nightly.json` 派生。配置 `dist_server` 或 `CJV_DIST_SERVER` 时，两份文件位于分发根下。来源优先级和部署契约见[配置](../configuration.md)与[内部分发源](../enterprise/distribution-server.md)。

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

安装通道名会记录跟随该通道的选择。以下命令都会将 STS 通道更新到当前最新版本；首次使用 `update sts` 时也会安装缺失的通道：

```bash
cjv install sts
cjv update sts
cjv update
```

`install sts` 再次执行时也会更新已有的 STS 通道。无参数 `update` 只更新已跟踪的通道及其交叉编译 SDK。升级会保留已选组件，成功后移除不再需要的旧 SDK、stdx 和文档。

明确安装版本（例如 `cjv install sts-1.2.0`）会保留该固定版本，`update` 不会替换它。渠道与固定版本拥有独立的 SDK、stdx、文档和组件记录。`cjv uninstall sts` 删除渠道安装及其跟随渠道的交叉 SDK；`cjv uninstall sts-1.2.0` 只删除固定版本。

首次安装通道时，默认工具链保存为 `sts` 等通道名。`default sts`、目录 override 或项目文件中的 `channel = "sts"` 会使用该通道当前的版本；明确的版本选择保持固定，项目文件不会被升级改写。

首次使用新布局时，cjv 在本地完成一次迁移：所有旧版本目录按固定版本保留，每个渠道原先隐含选择的最高已安装版本复制为独立渠道安装，文档、stdx 和组件记录一同复制。交叉 SDK 按所选主机的确切版本迁移；未安装配套版本时，仅保留现有固定目标安装。迁移无需联网，不改写默认工具链、override 或项目文件；具体版本继续固定，渠道选择继续可用。迁移完成后卸载的渠道不会在下次启动时重新创建。旧版本不再需要时可显式卸载。

安装记录位于各 SDK 的 `.cjv/toolchain.toml`，记录实际发行版本、平台和校验和。配置文件不需要维护安装别名表。同一发行包的 SDK 普通文件可通过硬链接节省空间，元数据始终独立；详情见[配置](../configuration.md)。

`cjv check` 查询 manifest，并比较已跟踪通道与各通道的最新版本，固定版本不会被报告为待升级：

```bash
cjv check
```
