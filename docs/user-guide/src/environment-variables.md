# 环境变量

cjv 的部分行为可以通过环境变量调整。环境变量适合临时覆盖、在 CI 中注入凭据，以及在不写入 `settings.toml` 的情况下改变默认行为。

下表列出面向用户的常用变量。除非另有说明，所有变量都在命令运行时读取，因此可以逐次调用临时设置：

```bash
CJV_LOG=debug cjv install lts
```

## 常用变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv` | 覆盖 cjv 的主目录（数据根目录）。必须是绝对路径，否则 cjv 会报错退出。该变量优先级高于 `settings.toml` 中持久化的 `home` 设置。 |
| `CJV_TOOLCHAIN` | 无 | 强制指定活跃工具链，覆盖所有其他解析方式（目录覆盖、工具链文件、默认工具链）。 |
| `CJV_LOG` | `warn` | 日志级别，可选 `debug`、`info`、`warn`、`error`。无法识别的值按 `warn` 处理。日志输出到标准错误（stderr）。 |
| `CJV_MAX_RETRIES` | `3` | 单次下载失败后的最大重试次数。取值需为非负整数，非法值将被忽略并回退到默认值。 |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP 下载的超时时间（秒）。取值需为正整数，非法值将被忽略并回退到默认值。 |
| `CJV_GITCODE_API_KEY` | 无 | GitCode API 访问令牌，用于查询和下载 nightly 工具链与组件。优先级高于 `settings.toml` 中持久化的令牌，且不会被写回磁盘。 |
| `CJV_NO_PATH_SETUP` | 无 | 设为 `1` 跳过首次安装时的 `PATH` 自动配置（适用于 CI 环境和集成测试）。其他值（包括未设置）不生效。 |
| `CANGJIE_STDX_PATH_DYNAMIC` | 由 cjv 注入 | 指向 `<CJV_HOME>/stdx/<tc>/dynamic`，仅当对应工具链安装了 `stdx` 组件时注入。通常无需手动设置。 |
| `CANGJIE_STDX_PATH_STATIC` | 由 cjv 注入 | 指向 `<CJV_HOME>/stdx/<tc>/static`，仅当对应工具链安装了 `stdx` 组件时注入。通常无需手动设置。 |

## 详细说明

### `CJV_HOME`

`CJV_HOME` 决定 cjv 存放工具链、组件、文档、下载缓存与设置文件的根目录，默认是用户主目录下的 `~/.cjv`。

它必须是绝对路径。相对路径会随当前工作目录变化而指向不同位置，导致从不同目录调用 cjv 时看到不同的安装集合，因此 cjv 会拒绝相对路径并报错退出。

主目录的解析顺序为（从高到低）：

1. `CJV_HOME` 环境变量
2. `settings.toml` 中持久化的 `home`（通过 `cjv set home <path>` 写入）
3. 默认值 `~/.cjv`

如需将主目录持久化而非每次设置环境变量，参见[配置](configuration.md)。`cjv show home` 会打印当前生效的主目录及其来源。

### `CJV_TOOLCHAIN`

`CJV_TOOLCHAIN` 位于工具链解析优先级的最顶端，会覆盖目录覆盖、工具链文件（`cangjie-sdk.toml`）和默认工具链。常用于临时切换工具链运行单条命令：

```bash
CJV_TOOLCHAIN=nightly cjc --version
```

完整的解析顺序详见[目标与覆盖](concepts/targets-overrides.md)。

### `CJV_LOG`

将日志级别调到 `debug` 可以观察下载、解析与代理执行的细节，便于排查问题：

```bash
CJV_LOG=debug cjv install lts
```

### `CJV_MAX_RETRIES` 与 `CJV_DOWNLOAD_TIMEOUT`

这两个变量用于在网络不稳定或镜像较慢时调整下载行为：

```bash
# 提高重试次数并延长超时（适用于慢速网络）
CJV_MAX_RETRIES=5 CJV_DOWNLOAD_TIMEOUT=600 cjv install sts
```

`CJV_MAX_RETRIES` 指失败后的重试次数，`CJV_DOWNLOAD_TIMEOUT` 以秒为单位。两者的非法值都会被忽略并回退到默认值。

### `CJV_GITCODE_API_KEY`

查询和下载 nightly 工具链及其组件需要 GitCode API 令牌。设置该环境变量可以在不把令牌写入 `settings.toml` 的情况下提供凭据，适合 CI 与部署场景：

```bash
CJV_GITCODE_API_KEY=your_token cjv install nightly
```

该环境变量的优先级高于持久化设置，且不会被写回磁盘。若希望长期保存令牌，改用：

```bash
cjv set gitcode-api-key <key>
```

关于 nightly 通道与 GitCode 的更多说明，参见[通道](concepts/channels.md)。

### `CJV_NO_PATH_SETUP`

首次安装工具链时，cjv 会把自身的 `bin` 目录加入 `PATH`，使代理命令（如 `cjc`、`cjpm`）立即可用。在 CI 环境或集成测试中，自动修改 `PATH` 往往不必要，可将该变量设为 `1` 跳过：

```bash
CJV_NO_PATH_SETUP=1 cjv install lts
```

只有值恰好为 `1` 时才会跳过，其他值不生效。

### `CANGJIE_STDX_PATH_DYNAMIC` 与 `CANGJIE_STDX_PATH_STATIC`

这两个变量由 cjv 在代理执行和运行时环境配置（`cjv exec` / `cjv envsetup`）中自动注入，分别指向 `stdx` 组件解压后的 `dynamic` 与 `static` 目录。仅当当前工具链安装了 `stdx` 组件时才会注入。

一般情况下无需手动设置它们，cjv 会确保仓颉编译器和构建工具能正确找到扩展库。关于 `stdx` 组件的安装与目录布局，参见[组件](concepts/components.md)；关于运行时环境的配置方式，参见[运行时环境](runtime-environment.md)。

## 高级与内部变量

以下变量面向特殊场景或由 cjv 内部使用，普通用户通常无需关心。

| 变量 | 说明 |
| --- | --- |
| `CJV_LANG` | 覆盖界面语言（如 `zh`、`en`、`ja`）。未设置时跟随系统区域设置。 |
| `CJV_ALLOW_INSECURE_MANIFEST` | 设为 `1` 时，允许从非回环（loopback）主机通过明文 HTTP 拉取工具链清单。默认要求 HTTPS，因为清单同时携带下载 URL 与其校验和。仅在信任的内部镜像场景使用，详见[从 URL 安装工具链](install-from-url.md)。 |
| `CJV_FALLBACK_SETTINGS` | 指定系统级后备设置文件的路径，用于在用户设置之外提供默认值（如企业镜像配置）。未设置时使用平台默认位置。 |
| `CJV_RECURSION_COUNT` | 仅内部使用。cjv 在代理执行时设置此变量以检测并阻止无限递归调用，用户不应手动设置。 |
