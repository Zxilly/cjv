# 代理与完全离线环境

## 仅使用企业代理

如果允许通过统一代理访问上游，不必建设内部分发源。为 cjv 进程设置 `HTTPS_PROXY` / `https_proxy`，并用 `NO_PROXY` / `no_proxy` 排除内部服务：

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv 不读取 `ALL_PROXY`。TLS 解密代理使用企业 CA 时，需要先把 CA 安装进操作系统信任库。完整变量说明见[网络代理](../network-proxies.md)。

慢速链路可以调整下载重试和整个请求的超时：

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

如果同时使用内部分发源，建议把 `dist_server` 主机加入 `NO_PROXY`，避免内部制品流量绕行外部代理。

## 完全离线部署

已经拿到 SDK 归档或解压目录时，不需要 manifest：

```bash
# 从本地归档物化安装；推荐始终提供批准的 SHA-256
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>

# 或引用已经解压的 SDK 目录
cjv toolchain link corp-sdk /opt/corp/cangjie-sdk

# stdx 可以从本地目录链接
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

本地归档或目录会创建 custom 工具链，项目中的 `cangjie-sdk.toml` 应使用相同的自定义名称。`stdx` 支持本地链接；`docs` 和 `stdx-docs` 当前不支持本地目录链接，应在终端进入隔离区前通过内部镜像安装好。更多归档格式说明见[从 URL 或本地归档安装工具链](../install-from-url.md)。

本地下载缓存不能替代分发 manifest。需要执行远程查询或更新时，仍应提供可达的内部 HTTPS 分发源；真正气隙环境应固定本地 custom 工具链，不运行远程查询命令。
