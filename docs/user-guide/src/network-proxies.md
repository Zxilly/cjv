# 网络代理

企业网络常常不允许直接访问外网，而要求通过代理服务器。cjv 会读取标准的代理环境变量，因此在这类网络里**无需任何 cjv 配置**——设好环境变量即可让 cjv 的所有下载（工具链、组件、清单、自更新）走代理。

## 设置代理

cjv 的下载都走 HTTPS，因此设置 `https_proxy` 通常就够了。不同系统与 shell 的命令略有差异：

- Linux / macOS（bash / zsh）：

  ```bash
  export https_proxy=http://proxy.example.com:8080
  ```

- Windows 命令提示符（cmd）：

  ```cmd
  set https_proxy=http://proxy.example.com:8080
  ```

- Windows PowerShell：

  ```powershell
  $env:https_proxy="http://proxy.example.com:8080"
  ```

代理 URL 支持 `http://`、`https://` 与 `socks5://` 三种方案。用 SOCKS5 代理时，把上面的值换成 `socks5://proxy.example.com:1080` 即可。

## 排除内网地址

`no_proxy` 用来列出**不**经过代理的主机，常用于让内部镜像或本地服务直连：

```bash
export no_proxy=localhost,127.0.0.1,mirror.corp.internal
```

例如用 `--mirror` 或自建镜像源安装工具链时，可以把镜像主机加入 `no_proxy`，让它绕过代理直连。

## 识别的变量

cjv 识别下列变量（大小写均可），与多数命令行工具一致：

| 变量 | 作用 |
| --- | --- |
| `https_proxy` / `HTTPS_PROXY` | HTTPS 请求使用的代理。cjv 的下载都是 HTTPS，这是最关键的一个 |
| `http_proxy` / `HTTP_PROXY` | HTTP 请求使用的代理 |
| `no_proxy` / `NO_PROXY` | 不经过代理的主机列表（逗号分隔） |

注意：cjv **不**识别 `ALL_PROXY` / `all_proxy`。如果你只设了 `ALL_PROXY`，请改设 `https_proxy`。

代理变量在 cjv 启动时从环境读取，因此请在运行 cjv **之前**于当前 shell 里设好（上面那种逐次 `https_proxy=… cjv …` 的前缀写法也可以）。

相关：cjv 自身读取的 `CJV_*` 变量见[环境变量](environment-variables.md)。
