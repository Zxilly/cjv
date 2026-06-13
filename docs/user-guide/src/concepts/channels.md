# 通道

通道（channel）是 cjv 对仓颉 SDK 发布流的命名。每个通道代表一条持续更新的发布线。安装一个通道时，cjv 会解析出该通道当前的最新版本并安装它。

cjv 支持三个内置通道：

| 通道      | 含义              | 下载来源                  | 额外要求                       |
| --------- | ----------------- | ------------------------- | ------------------------------ |
| `lts`     | 长期支持版        | 官方版本清单（manifest）  | 无                             |
| `sts`     | 短期支持版        | 官方版本清单（manifest）  | 无                             |
| `nightly` | 每日构建（预览版）| GitCode `nightly_build`   | 需要 `CJV_GITCODE_API_KEY`     |

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

裸版本号（如 `1.0.5`）只在 LTS / STS 的版本清单中查找，命中哪个通道就归属哪个通道，nightly 不参与裸版本号匹配。关于工具链命名的完整规则，参见[工具链](toolchains.md)。

## 下载来源的差异

三个通道安装的都是仓颉 SDK，区别在于取构建产物的位置不同。

### LTS / STS：官方版本清单

LTS 与 STS 的可用版本、下载地址和校验和来自一份官方维护的 JSON 版本清单（manifest）。cjv 内置了默认清单地址，安装时先拉取清单，再据此下载对应平台的 SDK 压缩包并校验 SHA-256。

清单地址可在 `~/.cjv/settings.toml` 中通过 `manifest_url` 覆盖（例如切换到镜像源），留空则恢复内置默认值。详见[配置](../configuration.md)。

### nightly：GitCode 每日构建仓库

nightly 不走版本清单。cjv 通过 GitCode 的发布 API 查询 `Cangjie/nightly_build` 仓库的最新发布，解析出 SDK 版本，再从该仓库的发布资产中下载对应平台的构建产物。

GitCode 的发布 API 需要鉴权，因此安装或检查 nightly 必须配置一个 GitCode API 访问令牌。未配置时，相关命令会失败并提示：

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

## 通道与组件来源

组件（component）的下载来源同样按通道区分。stdx、docs、stdx-docs 在 LTS / STS 与 nightly 下分别来自不同的发布仓库：

| 组件        | LTS / STS 来源             | nightly 来源         |
| ----------- | -------------------------- | -------------------- |
| `stdx`      | `cangjie_stdx` 发布        | `nightly_build` 发布 |
| `docs`      | `cangjie-docs-bundle` 发布 | `nightly_build` 发布 |
| `stdx-docs` | `cangjie_stdx` 发布        | `nightly_build` 发布 |

nightly 工具链的所有组件都从 `nightly_build` 仓库拉取，因此安装 nightly 组件同样依赖前面配置的 GitCode API 令牌。组件机制本身的说明见[组件](components.md)。

```bash
# 安装 nightly 时一并装上组件（组件也走 nightly_build 来源）
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

`cjv check` 会为已安装的通道型工具链查询是否有更新。nightly 的检查同样会调用 GitCode API，因此若你装有 nightly 工具链却未配置令牌，这一步会报错，LTS / STS 的检查不受影响。

```bash
cjv check
```
