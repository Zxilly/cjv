# 代理与完全离线环境

## 仅使用企业代理

代理模式通过 `HTTPS_PROXY` / `https_proxy` 访问上游，并用 `NO_PROXY` / `no_proxy` 直连内部服务：

```powershell
$env:HTTPS_PROXY = "http://proxy.corp.example:8080"
$env:NO_PROXY = "localhost,127.0.0.1,artifacts.corp.example"
cjv install lts-1.0.5
```

cjv 支持 `HTTP_PROXY`、`HTTPS_PROXY` 与 `NO_PROXY` 这一组标准变量。TLS 解密代理使用企业 CA 时，先把 CA 安装进操作系统信任库。完整变量说明见[网络代理](../network-proxies.md)。

慢速链路可以调整下载重试和整个请求的超时：

```powershell
$env:CJV_MAX_RETRIES = "5"
$env:CJV_DOWNLOAD_TIMEOUT = "600"
```

同时使用内部分发源时，把 `dist_server` 主机加入 `NO_PROXY`，让内部制品流量直接到达制品库。

## 完全离线部署

本地 SDK 归档或解压目录可以直接安装：

```bash
# 从本地归档物化安装；推荐始终提供批准的 SHA-256
cjv toolchain link corp-sdk ./cangjie-sdk.zip --sha256 <approved-sha256>

# 或引用已经解压的 SDK 目录
cjv toolchain link corp-sdk /opt/corp/cangjie-sdk

# stdx 可以从本地目录链接
cjv component link stdx /opt/corp/cangjie-stdx --toolchain corp-sdk
cjv default corp-sdk
```

本地归档或目录会创建 custom 工具链，项目中的 `cangjie-sdk.toml` 使用相同的自定义名称。`stdx` 支持本地链接；`docs` 和 `stdx-docs` 通过内部分发源预装。更多归档格式说明见[从 URL 或本地归档安装工具链](../install-from-url.md)。

远程查询和更新使用可达的内部 HTTPS manifest；气隙环境固定本地 custom 工具链并执行本地构建流程。
