维护旧项目时，你可能需要一套稳定的 SDK，同时用另一套 SDK 开发新项目。用项目目录决定工具链，可以避免每次切换工作时都修改全局环境。

下面使用两个并列目录 `project-a` 与 `project-b` 演示。它们应是已有的仓颉项目，且已经安装 cjv；示例中的 LTS、STS 需要在所用平台可用。`cjv pin` 的选项以[命令参考](https://cjv.zxilly.com/book/user-guide/zh-CN/command-reference.html)为准。

## 为两个项目分别固定版本

在第一个项目中，把 LTS 解析成一个具体发行版并写入项目文件：

```bash
cd project-a
cjv pin lts
cjv pin --check
cjc --version
```

`cjv pin` 写入 `cangjie-sdk.toml`，不安装 SDK。如果本机已有对应跟踪通道，它优先使用本机安装的发行版。首次运行 `cjc` 时，默认开启的自动安装会补齐缺失的固定版本。

然后进入并列的第二个项目，选择它需要的 STS：

```bash
cd ../project-b
cjv pin sts
cjv pin --check
cjc --version
```

打开两个生成的文件，确认它们包含各自解析后的完整发行版名称。团队已有指定版本时，直接把 `lts` 或 `sts` 替换为分发源提供的完整名称。将两个项目各自的 `cangjie-sdk.toml` 提交到版本控制。

## 通过目录切换验证选择结果

```bash
cd ../project-a
cjv show active
cjv which cjc
cjpm build

cd ../project-b
cjv show active
cjv which cjc
cjpm build
```

检查 `show active` 的工具链名称和来源是否对应当前项目，`which cjc` 是否指向预期 SDK。版本号取决于你固定的发行版，不应照抄文章中的示例输出。

固定版本与跟踪通道是独立的安装身份。即使你已经安装了 `lts`，第一次使用它解析出的固定版本时仍可能需要安装。后续更新跟踪通道不会自动改写项目文件。

## 临时试用另一套 SDK

要在 project-a 中测试 nightly，又不改动提交给团队的配置，可以只为单条命令指定工具链：

```bash
cjv exec +nightly cjc --version
cjv exec +nightly cjpm build
```

如果需要连续测试多个命令，也可以设置本机目录覆盖：

```bash
cjv override set nightly
cjv show active
cjpm build
cjv override unset
cjv show active
```

目录覆盖保存在本机设置中。完成后清除覆盖，再确认来源回到项目文件。夜间版本的可用性和兼容性取决于分发源，测试前保留项目原有的构建结果或使用独立工作目录。

## 版本不符合预期时检查什么

先运行 `cjv show active`，再检查命令行选择和 `CJV_TOOLCHAIN` 环境变量。它们的优先级高于项目文件。

目录配置按就近原则向上查找：同一目录里，本机覆盖优先于项目文件；子目录的项目文件则可以优先于父目录的覆盖。不同层级的项目文件不会合并。完整规则见[项目工具链](https://cjv.zxilly.com/book/user-guide/zh-CN/toolchain-file.html)。

项目依赖 stdx 或交叉目标时，把 `components`、`targets` 一起写入项目文件。显式指定其他工具链时，项目文件中的组件与目标要求不会自动附加到该选择上，测试环境也要准备相应依赖。

最后，在 CI 中执行 `cjv pin --check`，让浮动通道配置在构建前暴露出来。接入方法见 [GitHub Actions 实践](https://cjv.zxilly.com/blog/cangjie-github-actions/)。
