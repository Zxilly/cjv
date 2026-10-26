# 网站部署与边缘路由

`cjv` 官方网站同时部署在两个域名上，共享同一次 GitHub Actions 构建出的网站产物（包括落地页、中英文双语手册、安装脚本和 `/dl/` 独立安装器）：

- **`cjv.zxilly.dev`**：托管于 GitHub Pages，并由 Cloudflare Worker 处理首页地区个性化与地区跳转。
- **`cjv.zxilly.com`**：托管于腾讯云 EdgeOne Makers（全球加速含中国大陆可用区）。

边缘函数的源码结构与本地测试见 [deployment/README.md](../deployment/README.md)，文档站构建见开发指南中的 `docs/dev-guide/src/documentation.md`。

## 自动化部署工作流

### 静态站点与 EdgeOne 部署（`pages.yml`）

`.github/workflows/pages.yml` 在前端、文档或部署配置发生变化时自动运行，同时支持手动触发（`workflow_dispatch`）和发布工作流（`release.yml`）调用：

1. 构建 `web/` 前端、提取 14 个 `/dl/` 安装器、构建四份 mdBook 手册，并打包为同一份 `github-pages` 制品。
2. 将该制品发布到 GitHub Pages（`cjv.zxilly.dev`）。
3. 将同一份制品转换为 EdgeOne 预构建目录 `.edgeone` 并部署到腾讯云 EdgeOne（`cjv.zxilly.com`）。两个部署任务都必须成功。

### Cloudflare Worker 部署（`cloudflare.yml`）

Cloudflare Worker 由 `.github/workflows/cloudflare.yml` 独立部署。该工作流仅在 Worker 适配器、共享响应处理逻辑、构建工具、验证工具或工作流文件本身变化时触发。

由于 Worker 每次请求都会实时从 GitHub Pages 回源获取当前首页 HTML，日常修改前端文案或文档内容不需要重新部署 Worker。仅修改 Go 源码也不会触发网站部署，但正式发布工作流调用 `pages.yml` 时仍会更新 `/dl/` 安装器。

### 仓库变量与凭据要求

| 配置类型 | 名称 | 用途说明 |
| --- | --- | --- |
| Repository variable | `EDGEONE_PROJECT_NAME` | 已预先创建的 EdgeOne Makers 生产项目名称 |
| Repository secret | `EDGEONE_API_TOKEN` | 腾讯云中国站 Makers API Token |
| Repository variable | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare Worker 所属账户 ID |
| Repository secret | `CLOUDFLARE_API_TOKEN` | Cloudflare Worker 部署凭据（所需权限见 [deployment/README.md](../deployment/README.md)） |

> **注意：** 缺少 EdgeOne 项目或 Token 会使部署明确报错失败。请勿通过删除凭据来暂停部署；如需暂停部署，请直接修改工作流配置。

## EdgeOne 手动构建与部署

如需在本地手动部署 EdgeOne，请先从成功的网站工作流下载 `github-pages` 制品，依次解开外层 ZIP 和内部的 `artifact.tar`。解压出的目录必须包含 `index.html`、`book/`、`dl/`、`install.sh`、`install.ps1` 和 `404.html`。

假设 `./site` 是解压出的完整网站目录，在仓库根目录执行：

```sh
node deployment/build-edge.mjs edgeone ./site ./edgeone-output/.edgeone
npx --yes edgeone@1.6.44 login --site china --token "$EDGEONE_API_TOKEN"
npx --yes edgeone@1.6.44 makers deploy ./edgeone-output/.edgeone \
  --name "$EDGEONE_PROJECT_NAME" --env production --skip-ai-gateway-sync
node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/
```

> **注意：** 传给 `makers deploy` 的预构建目录名**必须**为 `.edgeone`。直接上传 `./site` 会绕过边缘函数；将预构建目录改为其他名称会让 CLI 将其当作普通静态上传，导致函数失效且静态资源路径错位。

- `deployment/build-edge.mjs` 会统一生成首页边缘函数和路由配置：首页按来访国家代码（`CN`）设置初始镜像开关，其他路径通过 origin hook 返回静态文件或 `404.html`。
- `--site china` 选择腾讯云中国站账户入口（与项目加速区域设置相互独立）。请先在控制台创建项目并确认加速区域，再配置 Actions 变量，防止 CLI 因找不到项目而意外新建项目。
- `--skip-ai-gateway-sync` 跳过静态网站不使用的 AI Gateway 同步。

## 域名配置与地区跳转规则

### EdgeOne 与 GitHub Pages 域名配置

- **`cjv.zxilly.com`（EdgeOne）**：在 Makers 控制台绑定自定义域名到生产环境，在 Cloudflare DNS 中添加指向控制台给定目标的 CNAME 记录，并将代理状态设为**仅 DNS（灰色云朵）**。包含中国大陆的加速区域要求域名已完成 ICP 备案；如需调整加速区域，应创建新项目并按控制台的域名释放状态迁移域名。
- **`cjv.zxilly.dev`（GitHub Pages + Cloudflare）**：保留 GitHub Pages 自定义域名与 HTTPS 配置，并在 Cloudflare DNS 中开启代理（橙色云朵）。请通过 Cloudflare Configuration Rules 为 `cjv.zxilly.dev` 单独设置 **Full (strict)** SSL 回源模式，避免 `Flexible` 模式与 GitHub Pages 的强制 HTTPS 跳转产生重定向死循环；不要为此修改整个 `zxilly.dev` DNS 区域的全局 SSL 模式。

### 中国大陆访问跳转规则

在 `cjv.zxilly.dev` 的 Cloudflare 入口配置 Redirect Rule，匹配表达式如下：

```text
(http.host eq "cjv.zxilly.dev" and ip.src.country eq "CN" and
 (http.request.uri.path in {"/" "/index.html"} or starts_with(http.request.uri.path, "/book/")))
```

- **重定向目标**：`concat("https://cjv.zxilly.com", http.request.uri.path)`，状态码 `302`，并勾选保留查询参数（Preserve query string）。
- **判断依据**：严格按来源 IP 国家代码（`CN`）判断，不按浏览器语言判断；`cjv.zxilly.com` 不配置反向地区跳转。
- **安装脚本豁免**：该规则仅匹配首页（`/`、`/index.html`）和文档（`/book/*`），保留 `.dev/install.sh`、`.dev/install.ps1` 和 `.dev/dl/*` 直接返回内容，因为已发布的 `curl -sSf ... | sh` 命令默认不跟随 `302` 重定向。

## 部署验证与应急回退

### 自动化与多出口验证

每次部署后，运行 `deployment/verify-site.mjs` 检查两个域名的首页、静态资源、双语文档、安装脚本、`/dl/` 下载入口和 `404` 响应；下载的二进制还应与同次发布的 `checksums.txt` 核对一致。

调整地区跳转规则后，请分别从中国大陆出口和境外出口验证：

- 访问 `.dev` 首页和深层文档（`/book/...`）时的 HTTP 状态码、`Location` 响应头与查询参数保留情况。
- 从两种出口访问 `.dev/install.sh` 和 `.dev/install.ps1` 时，均应直接返回 `200` 脚本内容而不发生重定向（客户端伪造国家请求头不能替代真实出口验证）。

### 故障回退步骤

1. 如果 `cjv.zxilly.com` 出现异常，先在 Cloudflare 中禁用上述中国大陆 `302` 重定向规则，确保用户可通过 `cjv.zxilly.dev` 正常访问。
2. 如果 Cloudflare Worker 或代理入口本身异常，再将 `cjv.zxilly.dev` 的 CNAME 切换回**仅 DNS** 直连 GitHub Pages。
3. 需要重新部署历史 `github-pages` 制品时，仍应使用当前最新的 `node deployment/build-edge.mjs` 构建步骤，避免带回已修复的旧版目录布局问题。

## 历史迁移记录与外部参考

### 2026-10-20 迁移记录

以下为 2026-10-20 当日的操作归档，当前生产状态请以 GitHub Actions、Makers 控制台及线上实时验证为准：

- 初始项目 `cjv-web`（`makers-9riya3l0zoze`），首次部署 ID `dpqt1qkz6d2l`（使用 `v0.4.1` 安装器），后续部署 `dpjy0k9ak9gn` 当时保留用于回退。
- 迁移后的全球加速项目为 `cjv-web-global`（`makers-nbvih0bn28ch`），当时验证的部署 ID 为 `dp8y8ezpjjzq`，加速区域为全球可用区（含中国大陆），CNAME 目标为 `cjv.zxilly.com.pages.dnsoe4.com`。
- 域名切换后已验证 HTTPS、四份双语文档、两份安装脚本、14 个平台安装器及缺失路径的 `404` 响应；Cloudflare Trace 验证了 `CN` 规则的 `302` 跳转，境外出口验证了 `.dev` 页面和脚本直返。

### 官方参考文档

- [EdgeOne Makers GitHub Actions 部署](https://pages.edgeone.ai/document/using-github-actions)
- [EdgeOne Makers CLI](https://pages.edgeone.ai/document/edgeone-cli)
- [EdgeOne Makers 构建输出与路由配置](https://pages.edgeone.ai/document/building-output-configuration)
- [EdgeOne Makers 域名及加速区域](https://cloud.tencent.com/document/product/1552/127403)
- [Cloudflare 按地区重定向规则](https://developers.cloudflare.com/rules/url-forwarding/examples/redirect-country-subdomains/)
- [Cloudflare 按请求配置 SSL 模式](https://developers.cloudflare.com/rules/configuration-rules/settings/)
