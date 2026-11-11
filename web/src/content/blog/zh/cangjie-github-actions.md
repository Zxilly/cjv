本地构建成功后，CI 仍可能因为 SDK 版本不同而失败。把版本声明提交到项目中，再让工作流读取它，可以减少开发机与构建机之间的差异。

下面是一份用于 Ubuntu GitHub 托管 runner 的工作流模板。前提是仓库根目录已有可构建、可测试的 cjpm 项目，以及固定发行版的 `cangjie-sdk.toml`；SDK 必须支持 runner 的操作系统和架构。请在自己的仓库验证工作流和依赖。

## 先提交项目工具链文件

在本地项目根目录执行：

```bash
cjv pin lts
cjv pin --check
```

如果项目需要其他通道或指定版本，替换 `lts`。检查生成的文件并提交它。`cjv pin --check` 不修改文件，也不访问网络，可以用来阻止浮动版本进入 CI。具体选择方式见[固定项目版本](https://cjv.zxilly.com/blog/pin-cangjie-project-versions/)。

## 添加构建与测试工作流

保存为 `.github/workflows/cangjie.yml`：

```yaml
name: Cangjie

on:
  push:
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - name: Install cjv
        shell: bash
        run: |
          curl -sSf https://cjv.zxilly.com/install.sh -o "$RUNNER_TEMP/install-cjv.sh"
          sh "$RUNNER_TEMP/install-cjv.sh" -y --default-toolchain none
          echo "$HOME/.cjv/bin" >> "$GITHUB_PATH"
      - name: Validate and prepare the project SDK
        shell: bash
        run: |
          cjv --version
          cjv pin --check
          cjc --version
          cjv show active
      - name: Build and test
        shell: bash
        run: |
          cjpm build
          cjpm test
```

`--default-toolchain none` 只安装管理器。下一步中的 `cjc --version` 从项目文件选择 SDK，并在默认开启的自动安装机制下准备缺失项。`GITHUB_PATH` 在后续步骤生效，所以安装和调用分成两个步骤。

模板每次获取当前稳定版 cjv；它固定的是项目 SDK，不是管理器本体。需要更严格地复现工具版本时，在安装脚本参数中加入经项目验证的 `--version` 发布标签，并按更新策略维护该标签。这里固定的 runner 标签也不代表系统镜像内容永远不变。

## 明确缓存的边界

先让无缓存工作流成功，再考虑缓存 `<CJV_HOME>/cache`，默认即 `~/.cjv/cache`。该目录用于可复用的版本清单和经过校验的下载归档；它与已安装 SDK 目录不是同一概念。

使用 `actions/cache` 时，缓存键至少包含 runner 系统、架构、项目工具链文件的哈希和缓存格式版本。更换分发源时也要区分缓存。不要在不同平台之间共用完整 SDK 安装目录。

缓存命中不保证离线构建：项目还可能需要组件、交叉目标或 cjpm 依赖。`cjv --offline` 约束 cjv 自身的请求，并不会禁止子进程联网。离线准备方法见[自动化文档](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html)。

## 根据失败位置排查

- **找不到 cjv 或 cjc**：检查安装步骤是否成功，以及 `GITHUB_PATH` 是否在前一步写入了正确目录。
- **版本校验失败**：检查当前工作目录是否有固定发行版的 `cangjie-sdk.toml`。单仓库多个项目时，为验证和构建步骤设置相同的 `working-directory`。
- **SDK 或组件下载失败**：查看具体下载错误，确认版本和平台组合在分发源中存在；重试前不要把校验失败的归档当作有效缓存。
- **编译器可以启动，构建或测试失败**：查看 cjpm 日志、系统依赖和项目代码。SDK 安装成功并不代表测试通过。

自托管 runner 还应隔离作业的 `CJV_HOME` 与设置文件，避免继承其他项目的目录覆盖或环境变量。完整配置参见[配置与网络](https://cjv.zxilly.com/book/user-guide/zh-CN/automation.html)。
