# 通道

通道（channel）是 cjv 对仓颉 SDK 发布流的命名。每个通道代表一条持续更新的发布线。安装一个通道时，cjv 会解析出该通道当前的最新版本并安装它。

cjv 支持三个内置通道：

| 通道      | 含义              | 下载来源                  | 额外要求                       |
| --------- | ----------------- | ------------------------- | ------------------------------ |
| `lts`     | 长期支持版        | 官方版本清单（manifest）  | —                              |
| `sts`     | 短期支持版        | 官方版本清单（manifest）  | —                              |
| `nightly` | 每日构建（预览版）| 默认 GitCode；也可由统一 manifest 提供 | 默认模式需要 `CJV_GITCODE_API_KEY` |

通道名大小写不敏感，`LTS`、`Lts`、`lts` 等价。

## 选哪个通道

`lts` 是长期支持版，版本相对稳定，迭代节奏慢，适合生产构建以及对兼容性敏感的项目。如果你不确定该用哪个，从 `lts` 开始。

`sts` 是短期支持版，比 LTS 更新得快，能更早拿到新特性，但维护周期短。它适合希望跟进语言演进、又不想用每日构建的场景。

`nightly` 是每日构建，包含最新但尚未稳定的改动，可能随时变化或回归。它适合尝鲜、复现 upstream 行为，或为 SDK 本体提 bug。

```bash
# 安装某个通道的最新版本
cjv install lts
cjv install sts
cjv install nightly
```

## 通道与版本名

把通道名直接交给 `cjv install`，等同于安装该通道的最新版本。cjv 会先解析出具体版本号，再以 `<通道>-<版本>` 的形式落盘。例如安装 `lts` 可能得到一个名为 `lts-1.0.5` 的已安装工具链。

你也可以把版本写死，跳过解析最新这一步：

```bash
# 安装指定通道的指定版本
cjv install lts-1.0.5
cjv install sts-1.1.0-beta.23

# 仅给出裸版本号，cjv 会跨 LTS / STS 查找该版本所属的通道
cjv install 1.0.5
```

裸版本号（如 `1.0.5`）的搜索空间是 LTS / STS 版本清单，命中哪个通道就归属哪个通道。nightly 使用带通道前缀的版本名。关于工具链命名的完整规则，参见[工具链](toolchains.md)。

## 下载来源

三个通道安装的都是仓颉 SDK，区别在于取构建产物的位置不同。

### LTS / STS：官方版本清单

LTS 与 STS 的可用版本、下载地址和校验和来自一份官方维护的 JSON 版本清单（manifest）。cjv 内置了默认清单地址，安装时先拉取清单，再据此下载对应平台的 SDK 压缩包并校验 SHA-256。

清单地址可在 `~/.cjv/settings.toml` 中通过 `manifest_url` 覆盖（例如切换到镜像源），留空则恢复内置默认值。详见[配置](../configuration.md)。

### nightly：默认 GitCode，企业源使用统一 manifest

兼容模式通过 GitCode 发布 API 查询 `Cangjie/nightly_build` 仓库的最新发布，解析 SDK 版本，再下载对应平台的 Release 资产。

GitCode 的发布 API 使用访问令牌鉴权。兼容模式下安装或检查浮动的 `nightly` 时配置令牌；令牌缺失会返回以下提示：

```text
查询 nightly 版本需要 GitCode API 密钥。请通过以下命令设置: cjv set gitcode-api-key <your-token>
```

配置令牌有两种方式，环境变量优先于持久化设置：

```bash
# 方式一：写入设置（持久保存在 ~/.cjv/settings.toml）
cjv set gitcode-api-key <your-token>

# 方式二：通过环境变量提供（优先级更高，适合 CI）
export CJV_GITCODE_API_KEY=<your-token>
```

关于 `CJV_GITCODE_API_KEY` 的完整说明见[环境变量](../environment-variables.md)，关于 `cjv set` 见[配置](../configuration.md)。

配置 `dist_server` 或 `CJV_DIST_SERVER` 后，三个通道共用 `<dist_server>/versions.json`。nightly 的最新版本、精确版本、平台 SDK 和组件都由 manifest 描述；缺少内容返回明确错误。部署契约见[内部分发源](../enterprise/distribution-server.md)。

## 通道与组件来源

默认模式下，stdx、docs、stdx-docs 在 LTS / STS 与 nightly 下分别来自不同的发布仓库：

| 组件        | LTS / STS 来源             | nightly 来源         |
| ----------- | -------------------------- | -------------------- |
| `stdx`      | `cangjie_stdx` 发布        | `nightly_build` 发布 |
| `docs`      | `cangjie-docs-bundle` 发布 | `nightly_build` 发布 |
| `stdx-docs` | `cangjie_stdx` 发布        | `nightly_build` 发布 |

统一企业分发源下，三个通道的组件都使用 manifest 中声明的 URL，并可携带 SHA-256。默认 GitCode nightly 的组件仍按 `nightly_build` Release 布局下载。组件机制本身的说明见[组件](components.md)。

```bash
# 安装 nightly 时一并装上组件
cjv install nightly -c stdx,docs
```

## 在工具链文件中指定通道

项目可以在 `cangjie-sdk.toml` 的 `channel` 字段声明所需通道，让协作者拿到代码后自动使用同一通道：

```toml
[toolchain]
channel = "lts"
```

`channel` 既可以是通道名（`lts` / `sts` / `nightly`），也可以是带版本的工具链名（如 `lts-1.0.5`）。完整字段语义见[工具链文件](../toolchain-file.md)。

## 检查更新

`cjv check` 会为已安装的通道型工具链查询是否有更新。兼容模式的 nightly 检查调用 GitCode API；统一企业分发源让三个通道查询同一 manifest。

```bash
cjv check
```
