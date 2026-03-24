# cjv - Cangjie Version Manager

[English](README.EN.md) | 中文

[仓颉](https://cangjie-lang.cn/)编程语言 SDK 的工具链管理器。

cjv 管理多个仓颉 SDK 安装、处理版本切换，并提供 SDK 工具的透明代理执行。

## 安装

### 从源码编译

```bash
go install github.com/Zxilly/cjv/cmd/cjv@latest
```

### 从发布二进制文件

从 [Releases](https://github.com/Zxilly/cjv/releases) 页面下载适合你平台的二进制文件，并将其放入 PATH 中。

## 快速开始

```bash
# 安装最新 LTS 工具链
cjv install lts

# 设为默认
cjv default lts

# 验证安装
cjv show

# 使用指定工具链运行命令
cjv run sts cjc --version
```

## 命令

| 命令                                                | 说明                               |
| --------------------------------------------------- | ---------------------------------- |
| `cjv install <toolchain>`                           | 安装仓颉 SDK 工具链                |
| `cjv uninstall <toolchain>`                         | 卸载工具链                         |
| `cjv update [toolchain]`                            | 更新已安装的工具链                 |
| `cjv default [toolchain]`                           | 设置或显示默认工具链               |
| `cjv show`                                          | 显示活跃和已安装的工具链           |
| `cjv show active`                                   | 显示当前活跃的工具链               |
| `cjv show installed`                                | 列出已安装的工具链                 |
| `cjv show home`                                     | 显示 CJV_HOME 路径                 |
| `cjv run <toolchain> <command> [args...]`           | 使用指定工具链运行命令             |
| `cjv which <command>`                               | 显示活跃工具链中 SDK 工具的路径    |
| `cjv check`                                         | 检查可用更新（不安装）             |
| `cjv override set <toolchain>`                      | 为当前目录设置工具链覆盖           |
| `cjv override unset`                                | 移除当前目录的工具链覆盖           |
| `cjv override list`                                 | 列出所有目录覆盖                   |
| `cjv toolchain list`                                | 列出已安装的工具链                 |
| `cjv toolchain link <name> <path>`                  | 将自定义工具链链接到本地目录       |
| `cjv toolchain uninstall <name>`                    | 卸载工具链                         |
| `cjv set auto-self-update <enable\|disable\|check>` | 设置自动自更新行为                 |
| `cjv set auto-install <true\|false>`                | 设置代理模式下缺失工具链的自动安装 |
| `cjv set gitcode-api-key <key>`                     | 设置 GitCode API 访问令牌（nightly 构建需要） |
| `cjv self update`                                   | 更新 cjv 到最新版本                |
| `cjv self uninstall`                                | 卸载 cjv 及所有已安装的工具链      |
| `cjv self clean-cache`                              | 清理下载缓存                       |

## 工具链解析

cjv 按以下优先级顺序解析活跃工具链（从高到低）：

1. `CJV_TOOLCHAIN` 环境变量
2. 目录覆盖（通过 `cjv override set` 设置）
3. 工具链文件（当前目录或父目录中的 `cangjie-sdk.toml`）
4. 默认工具链（通过 `cjv default` 设置）

## 代理模式

当直接调用 SDK 工具（如 `cjc`、`cjpm`）时，cjv 会透明地将调用代理到相应的工具链。代理符号链接在安装时创建在 cjv 的 bin 目录中。

如果设置中启用了 `auto_install` 且解析到的工具链未安装，cjv 会在代理前自动安装它。

## 环境变量

| 变量                   | 说明                                                   |
| ---------------------- | ------------------------------------------------------ |
| `CJV_HOME`             | 覆盖默认主目录（默认: `~/.cjv`）                       |
| `CJV_TOOLCHAIN`        | 强制指定工具链，覆盖所有其他解析方式                   |
| `CJV_LOG`              | 设置日志级别: `debug`、`info`、`warn`（默认）、`error` |
| `CJV_MAX_RETRIES`      | 下载失败最大重试次数（默认: `3`）                      |
| `CJV_DOWNLOAD_TIMEOUT` | HTTP 下载超时秒数（默认: `180`）                       |
| `CJV_GITCODE_API_KEY`  | GitCode API 访问令牌，用于查询和下载 nightly 工具链    |
| `CJV_NO_PATH_SETUP`    | 设为 `1` 跳过首次安装时的 PATH 自动配置                |

## 目录结构

```
~/.cjv/
  bin/            # 代理符号链接和 cjv 二进制文件
  toolchains/     # 已安装的 SDK 工具链
  downloads/      # 下载的 SDK 归档文件（缓存）
  settings.toml   # 用户设置
```

## 配置

设置存储在 `~/.cjv/settings.toml` 中，可通过 `cjv set` 命令修改。

## 许可证

Apache-2.0。详见 [LICENSE](LICENSE)。

## 致谢

cjv 的设计灵感来源于 [rustup](https://github.com/rust-lang/rustup)