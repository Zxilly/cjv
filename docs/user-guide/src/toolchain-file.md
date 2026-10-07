# 项目工具链

把 `cangjie-sdk.toml` 放在项目根目录并提交到版本控制，即可让项目成员使用同一套工具链声明：

```toml
[toolchain]
channel = "lts-1.0.5"
components = ["stdx"]
```

版本是示例；完整名称格式见[工具链与版本](concepts/toolchains.md)。

## 选择顺序

1. 命令中显式指定的工具链，例如全局 `+name` 或命令支持的 `--toolchain`。
2. `CJV_TOOLCHAIN` 环境变量。
3. 从当前目录向根目录逐层查找：每层先检查目录覆盖，再检查 `cangjie-sdk.toml`。找到后停止。
4. `cjv default` 设置的默认工具链。

因此，同一目录的覆盖优先于项目文件，子目录的项目文件优先于父目录的覆盖。不同目录的文件不会合并。`cjv show active` 显示最终选择及来源。

`cjv run <toolchain> <command>` 始终使用参数中的工具链。显式选择或环境变量生效时，项目文件中的组件和目标要求不会附加到该选择。

## 字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `channel` | string | 通道、版本选择器或已注册的自定义名称 |
| `path` | string | SDK 的绝对路径，与 `channel` 互斥 |
| `components` | string[] | 需要的 `stdx`、`docs`、`stdx-docs`，默认空 |
| `targets` | string[] | 需要的交叉 SDK 后缀，例如 `ohos`，默认空 |

`channel` 和 `path` 必须且只能指定一个。直接引用外部 SDK 时，组件由 SDK 所有者维护，`components` 与 `targets` 不参与自动安装：

```toml
[toolchain]
path = 'C:\SDKs\cangjie'
```

`targets` 只接受后缀，不接受 `linux-x64-ohos` 这样的完整平台名。大小写和下划线会被规范化，逗号分隔项会展开，重复项会去重；空目标会报错。

```toml
[toolchain]
channel = "sts"
components = ["stdx", "docs"]
targets = ["ohos", "android"]
```

这个声明准备宿主 SDK、宿主组件和目标 SDK。编译时怎样选择目标环境见[交叉编译](cross-compilation.md)。

## 自动安装与配置错误

`auto_install` 默认开启。运行 `cjc`、`cjpm` 等代理命令时，cjv 会补齐所选项目文件声明的缺失项。关闭后，缺失项会让命令报错：

```bash
cjv set auto-install false
```

未知键会打印警告，已识别字段继续解析。TOML 语法错误、未知组件，以及缺少有效 `channel` 或 `path` 都会报错。空文件也会报错；要恢复上级目录或默认选择，请删除文件。

## 本机目录覆盖

```bash
cjv override set nightly
cjv override set lts --path /path/to/project
cjv override list
cjv override unset
cjv override unset --nonexistent
```

覆盖保存在用户设置中，适用于指定目录及其子目录。`--nonexistent` 清除已不存在目录的覆盖。
