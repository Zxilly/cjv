# 环境变量

环境变量在启动 cjv 前设置，用于当前进程的临时覆盖：

```bash
CJV_LOG=debug cjv show active
```

```powershell
$env:CJV_LOG = "debug"
cjv show active
```

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| `CJV_HOME` | `~/.cjv`，或设置中的 `home` | 数据目录，必须为绝对路径；不改变用户设置文件位置 |
| `CJV_TOOLCHAIN` | 无 | 指定工具链名或 SDK 绝对路径；优先于目录和默认设置 |
| `CJV_DIST_SERVER` | 无 | 覆盖[工具链分发根](enterprise/distribution-server.md) |
| `CJV_LOG` | `warn` | `debug`、`info`、`warn`、`error`，未知值回退为 `warn`；输出到 stderr |
| `CJV_MAX_RETRIES` | `3` | 下载失败后的最大重试次数，非负整数 |
| `CJV_DOWNLOAD_TIMEOUT` | `180` | HTTP 下载超时秒数，正整数 |
| `CJV_NO_PATH_SETUP` | 无 | 恰好为 `1` 时跳过自动 PATH 配置 |
| `CJV_LANG` | 系统区域设置 | 覆盖界面语言，如 `zh`、`en`、`ja` |
| `CJV_FALLBACK_SETTINGS` | [系统配置路径](enterprise/index.md) | 指定后备设置文件 |
| `CJV_ALLOW_INSECURE_MANIFEST` | 无 | 为 `1` 时允许非回环主机通过明文 HTTP 提供清单 |

重试次数和超时的非法值回退到默认值。清单同时提供制品 URL 与校验和，默认要求 HTTPS；本地回环测试服务器不受该限制。

安装 stdx 后，cjv 为所选 SDK 的运行环境注入 `CANGJIE_STDX_PATH_DYNAMIC` 和 `CANGJIE_STDX_PATH_STATIC`，分别指向组件的 `dynamic/` 与 `static/`，通常无需手动设置。

HTTP(S) 代理变量见[网络代理](network-proxies.md)。`CJV_UPDATE_ROOT` 属于安装脚本，用于选择 cjv 本体的首次下载地址；它不覆盖已安装 cjv 的自更新源。
